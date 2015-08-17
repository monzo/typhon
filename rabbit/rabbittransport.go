package rabbit

import (
	"fmt"
	"os"
	"sync"
	"time"

	log "github.com/cihub/seelog"
	uuid "github.com/nu7hatch/gouuid"
	"github.com/streadway/amqp"
	"gopkg.in/tomb.v2"

	"github.com/mondough/typhon/errors"
	"github.com/mondough/typhon/message"
	"github.com/mondough/typhon/transport"
)

const (
	connectTimeout  = 30 * time.Second
	chanSendTimeout = 10 * time.Second
	respondTimeout  = 10 * time.Second
)

var (
	ErrCouldntConnect   = errors.InternalService("Could not connect to RabbitMQ")
	ErrDeliveriesClosed = errors.InternalService("Delivery channel closed")
	ErrNoReplyTo        = errors.BadRequest("Request does not have appropriate X-Rabbit-ReplyTo header")
)

type rabbitTransport struct {
	tomb          *tomb.Tomb
	connM         sync.RWMutex                       // protects conn + connReady
	conn          *RabbitConnection                  // underlying connection
	connReady     chan struct{}                      // swapped along with conn (reconnecting)
	replyQueue    string                             // message reply queue name
	inflightReqs  map[string]chan<- message.Response // correlation id: response chan
	inflightReqsM sync.Mutex                         // protects inflightReqs
	listeners     map[string]*tomb.Tomb              // service name: tomb
	listenersM    sync.RWMutex                       // protects listeners
}

// run starts the asynchronous run-loop connecting to RabbitMQ
func (t *rabbitTransport) run() {
	initConn := func() *RabbitConnection {
		conn := NewRabbitConnection()
		t.connM.Lock()
		defer t.connM.Unlock()
		t.conn = conn
		select {
		case <-t.connReady:
			// Only swap connReady if it's already closed
			t.connReady = make(chan struct{})
		default:
		}
		return conn
	}
	conn := initConn()

	t.tomb.Go(func() error {
		defer func() {
			t.killListeners()
			conn.Close()
			log.Info("[Typhon:RabbitTransport] Dead; connection closed")
		}()

	runLoop:
		for {
			log.Info("[Typhon:RabbitTransport] Run loop connecting…")
			select {
			case <-t.tomb.Dying():
				return nil

			case <-conn.Init():
				log.Info("[Typhon:RabbitTransport] Run loop connected")
				t.listenReplies()

				select {
				case <-t.tomb.Dying():
					// Do not loop again
					return nil
				default:
					conn.Close()
					conn = initConn()
					continue runLoop
				}

			case <-time.After(connectTimeout):
				log.Criticalf("[Typhon:RabbitTransport] Run loop timed out after %s waiting to connect",
					connectTimeout.String())
				return ErrCouldntConnect
			}
		}
	})
}

// deliveryChan returns the name of a delivery channel for a given service
func (t *rabbitTransport) deliveryChan(serviceName string) string {
	return serviceName
}

func (t *rabbitTransport) Tomb() *tomb.Tomb {
	return t.tomb
}

func (t *rabbitTransport) connection() *RabbitConnection {
	t.connM.RLock()
	defer t.connM.RUnlock()
	return t.conn
}

func (t *rabbitTransport) Ready() <-chan struct{} {
	t.connM.RLock()
	defer t.connM.RUnlock()
	return t.connReady
}

func (t *rabbitTransport) killListeners() {
	t.listenersM.RLock()
	ts := make([]*tomb.Tomb, 0, len(t.listeners))
	for _, t := range t.listeners {
		t.Killf("Listeners killed")
		ts = append(ts, t)
	}
	t.listenersM.RUnlock()
	for _, t := range ts {
		t.Wait()
	}
}

func (t *rabbitTransport) StopListening(serviceName string) bool {
	t.listenersM.RLock()
	tm, ok := t.listeners[t.deliveryChan(serviceName)]
	if ok {
		tm.Killf("Stopped listening")
	}
	t.listenersM.RUnlock()
	if ok {
		tm.Wait()
		return true
	}
	return false
}

func (t *rabbitTransport) Listen(serviceName string, rc chan<- message.Request) error {
	tm := &tomb.Tomb{}
	cn := t.deliveryChan(serviceName)
	t.listenersM.Lock()
	if _, ok := t.listeners[cn]; ok {
		t.listenersM.Unlock()
		return transport.ErrAlreadyListening
	}
	t.listeners[cn] = tm
	t.listenersM.Unlock()

	// Used to convey a connection error to the caller (we block until listening has begun)
	errChan := make(chan error)

	tm.Go(func() error {
		timeout := time.NewTimer(connectTimeout)
		defer func() {
			timeout.Stop()
			t.listenersM.Lock()
			defer t.listenersM.Unlock()
			delete(t.listeners, cn)
			close(rc)
			close(errChan)
			log.Debugf("[Typhon:RabbitTransport] Listener %s stopped", cn)
		}()

		select {
		case <-t.tomb.Dying():
			return nil
		case <-tm.Dying():
			return nil
		case <-timeout.C:
			errChan <- transport.ErrTimeout
			return nil
		case <-t.Ready():
		}

		deliveryChan, rabbitChannel, err := t.connection().Consume(cn)
		if err != nil {
			log.Warnf("[Typhon:RabbitTransport] Failed to consume from %s: %v", cn, err)
			errChan <- err
			return nil
		}
		defer rabbitChannel.Close()
		errChan <- nil
		log.Infof("[Typhon:RabbitTransport] Listening on %s…", cn)

		for {
			select {
			case <-t.tomb.Dying():
				return nil

			case <-tm.Dying():
				return nil

			case delivery, ok := <-deliveryChan:
				if !ok {
					log.Warnf("[Typhon:RabbitTransport] Delivery channel closed; stopping listener %s", cn)
					return nil
				}
				go t.handleReqDelivery(delivery, rc)
			}
		}
	})
	return <-errChan
}

func (t *rabbitTransport) logId(delivery amqp.Delivery) string {
	return fmt.Sprintf("%s[%s]", delivery.RoutingKey, delivery.CorrelationId)
}

func (t *rabbitTransport) Respond(req message.Request, rsp message.Response) error {
	headers := rsp.Headers()
	headers["Content-Encoding"] = "response"
	headers["Service"] = rsp.Service()
	headers["Endpoint"] = rsp.Endpoint()
	headers["Origin-Service"] = rsp.OriginService()
	headers["Origin-Endpoint"] = rsp.OriginEndpoint()

	timeout := time.NewTimer(respondTimeout)
	defer timeout.Stop()
	select {
	case <-t.Ready():
		timeout.Stop()
	case <-t.tomb.Dying():
		return tomb.ErrDying
	case <-timeout.C:
		return transport.ErrTimeout
	}

	return t.connection().Publish("", req.Headers()["X-Rabbit-ReplyTo"], amqp.Publishing{
		CorrelationId: rsp.Id(),
		Timestamp:     time.Now().UTC(),
		Body:          rsp.Payload(),
		Headers:       headersToTable(headers),
	})
}

func (t *rabbitTransport) Send(req message.Request, _timeout time.Duration) (message.Response, error) {
	id := req.Id()
	if id == "" {
		_uuid, err := uuid.NewV4()
		if err != nil {
			log.Errorf("[Typhon:RabbitTransport] Failed to generate request uuid: %v", err)
			return nil, err
		}
		req.SetId(_uuid.String())
	}

	rspQueue := req.Id()
	defer func() {
		t.inflightReqsM.Lock()
		delete(t.inflightReqs, rspQueue)
		t.inflightReqsM.Unlock()
	}()
	rspChan := make(chan message.Response, 1)
	t.inflightReqsM.Lock()
	t.inflightReqs[rspQueue] = rspChan
	t.inflightReqsM.Unlock()

	timeout := time.NewTimer(_timeout)
	defer timeout.Stop()

	headers := req.Headers()
	headers["Content-Encoding"] = "request"
	headers["Service"] = req.Service()
	headers["Endpoint"] = req.Endpoint()
	headers["Origin-Service"] = req.OriginService()
	headers["Origin-Endpoint"] = req.OriginEndpoint()

	select {
	case <-t.Ready():
	case <-timeout.C:
		log.Warnf("[Typhon:RabbitTransport] Timed out after %s waiting for ready", _timeout.String())
		return nil, transport.ErrTimeout
	}

	if err := t.connection().Publish(Exchange, req.Service(), amqp.Publishing{
		CorrelationId: req.Id(),
		Timestamp:     time.Now().UTC(),
		Body:          req.Payload(),
		ReplyTo:       t.replyQueue,
		Headers:       headersToTable(headers),
	}); err != nil {
		log.Errorf("[Typhon:RabbitTransport] Failed to publish: %v", err)
		return nil, err
	}

	select {
	case rsp := <-rspChan:
		return rsp, nil
	case <-timeout.C:
		log.Warnf("[Typhon:RabbitTransport] Timed out after %s waiting for response to %s", _timeout.String(),
			req.Id())
		return nil, transport.ErrTimeout
	}
}

func (t *rabbitTransport) listenReplies() error {
	if err := t.connection().Channel.DeclareReplyQueue(t.replyQueue); err != nil {
		log.Criticalf("[Typhon:RabbitTransport] Failed to declare reply queue %s: %v", t.replyQueue, err)
		os.Exit(1)
		return err
	}

	deliveries, err := t.connection().Channel.ConsumeQueue(t.replyQueue)
	if err != nil {
		log.Criticalf("[Typhon:RabbitTransport] Failed to consume from reply queue %s: %v", t.replyQueue, err)
		os.Exit(1)
		return err
	}

	log.Debugf("[Typhon:RabbitTransport] Listening for replies on %s…", t.replyQueue)
	t.connM.RLock()
	readyC := t.connReady
	t.connM.RUnlock()
	select {
	case <-readyC:
		// Make sure not to close the channel if it's already closed
	default:
		close(readyC)
	}

	for {
		select {
		case delivery, ok := <-deliveries:
			if !ok {
				log.Warnf("[Typhon:RabbitTransport] Delivery channel %s closed", t.replyQueue)
				return ErrDeliveriesClosed
			}
			go t.handleRspDelivery(delivery)

		case <-t.tomb.Dying():
			log.Info("[Typhon:RabbitTransport] Reply listener terminating (tomb death)")
			return tomb.ErrDying
		}
	}
}

func (t *rabbitTransport) deliveryToMessage(delivery amqp.Delivery, msg message.Message) {
	msg.SetId(delivery.CorrelationId)
	msg.SetHeaders(tableToHeaders(delivery.Headers))
	msg.SetHeader("X-Rabbit-ReplyTo", delivery.ReplyTo)
	msg.SetPayload(delivery.Body)
	switch service := delivery.Headers["Service"].(type) {
	case string:
		msg.SetService(service)
	}
	switch endpoint := delivery.Headers["Endpoint"].(type) {
	case string:
		msg.SetEndpoint(endpoint)
	}
	switch originService := delivery.Headers["Origin-Service"].(type) {
	case string:
		msg.SetOriginService(originService)
	}
	switch originEndpoint := delivery.Headers["Origin-Endpoint"].(type) {
	case string:
		msg.SetOriginEndpoint(originEndpoint)
	}
}

func (t *rabbitTransport) handleReqDelivery(delivery amqp.Delivery, reqChan chan<- message.Request) {
	logId := t.logId(delivery)
	enc := delivery.Headers["Content-Encoding"].(string)
	switch enc {
	case "request":
		req := message.NewRequest()
		t.deliveryToMessage(delivery, req)

		timeout := time.NewTimer(chanSendTimeout)
		defer timeout.Stop()
		select {
		case reqChan <- req:
		case <-timeout.C:
			log.Errorf("[Typhon:RabbitTransport] Could not deliver request %s after %s: receiving channel is full",
				logId, chanSendTimeout.String())
		}

	default:
		log.Debugf("[Typhon:RabbitTransport] Cannot handle Content-Encoding \"%s\" as request for %s", enc, logId)
	}
}

func (t *rabbitTransport) handleRspDelivery(delivery amqp.Delivery) {
	logId := t.logId(delivery)

	enc := delivery.Headers["Content-Encoding"].(string)
	switch enc {
	case "response":
		rsp := message.NewResponse()
		t.deliveryToMessage(delivery, rsp)

		t.inflightReqsM.Lock()
		rspChan, ok := t.inflightReqs[rsp.Id()]
		delete(t.inflightReqs, rsp.Id())
		t.inflightReqsM.Unlock()
		if !ok {
			log.Warnf("[Typhon:RabbitTransport] Could not match response %s to channel", logId)
			return
		}

		timeout := time.NewTimer(chanSendTimeout)
		defer timeout.Stop()
		select {
		case rspChan <- rsp:
		case <-timeout.C:
			log.Errorf("[Typhon:RabbitTransport] Could not deliver response %s after %s: receiving channel is full",
				logId, chanSendTimeout.String())
		}

	default:
		log.Errorf("[Typhon:RabbitTransport] Cannot handle Content-Encoding \"%s\" as response for %s", enc, logId)
	}
}

func NewTransport() transport.Transport {
	rt := &rabbitTransport{
		tomb:         new(tomb.Tomb),
		inflightReqs: make(map[string]chan<- message.Response),
		listeners:    make(map[string]*tomb.Tomb),
		connReady:    make(chan struct{}),
	}

	if uid, err := uuid.NewV4(); err != nil {
		log.Criticalf("[Typhon:RabbitTransport] Failed to create UUID for reply queue: %v", err)
		os.Exit(1) // TODO: Is this really necessary?
	} else {
		rt.replyQueue = fmt.Sprintf("replyTo-%s", uid.String())
	}
	rt.run()
	return rt
}
