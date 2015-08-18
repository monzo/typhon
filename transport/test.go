package transport

import (
	"math/rand"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/mondough/typhon/message"
)

const testService = "service.example"

// TransportTestSuite is a test suite which can be extended to test any Transport's basic functionality. Provide the
// Transport function and run it.
type TransportTestSuite struct {
	suite.Suite
	Transport Transport
}

// Actual tests

func (suite *TransportTestSuite) SetupTest() {
	trans := suite.Transport
	select {
	case <-trans.Ready():
	case <-trans.Tomb().Dead():
		panic("transport dead before ready")
	}
}

func (suite *TransportTestSuite) TearDownTest() {
	trans := suite.Transport
	trans.Tomb().Killf("Test ending")
	trans.Tomb().Wait()
	suite.Transport = nil
}

// TestSendReceive verifies an end-to-end flow over a transport: binding to receive requests, sending a request to
// oneself as a client, and receiving a reply.
func (suite *TransportTestSuite) TestSendReceive() {
	trans := suite.Transport
	inboundChan := make(chan message.Request, 1)
	trans.Listen(testService, inboundChan)

	go func() {
		select {
		case req := <-inboundChan:
			suite.Assert().NotNil(req)
			suite.Assert().Equal("ping", string(req.Payload()))
			suite.Assert().Equal("Shut up and take my money!", req.Headers()["X-Fry"])
			suite.Assert().Equal(testService, req.Service())
			suite.Assert().Equal("foo", req.Endpoint())

			rsp := message.NewResponse()
			rsp.SetId(req.Id())
			rsp.SetService(req.Service())
			rsp.SetEndpoint(req.Endpoint())
			rsp.SetPayload([]byte("pong"))
			suite.Assert().NoError(trans.Respond(req, rsp))

		case <-trans.Tomb().Dying():
		}
	}()

	req := message.NewRequest()
	req.SetService(testService)
	req.SetEndpoint("foo")
	req.SetPayload([]byte("ping"))
	req.SetId("1")
	req.SetHeader("X-Fry", "Shut up and take my money!")
	rsp, err := trans.Send(req, time.Second)
	suite.Assert().NoError(err)
	suite.Assert().NotNil(rsp)
	suite.Assert().Equal(req.Service(), rsp.Service())
	suite.Assert().Equal(req.Endpoint(), rsp.Endpoint())
	suite.Assert().Equal(req.Id(), rsp.Id())
	suite.Assert().Equal("pong", string(rsp.Payload()))
}

// TestAlreadyListening checks that duplicate listeners can't be registered
func (suite *TransportTestSuite) TestAlreadyListening() {
	trans := suite.Transport
	inboundChan := make(chan message.Request, 1)
	suite.Assert().NoError(trans.Listen(testService, inboundChan))
	suite.Assert().Equal(ErrAlreadyListening, trans.Listen(testService, inboundChan))
}

// TestSendReceiveParallel sends a bunch of requests in parallel and checks that the responses match correctly
func (suite *TransportTestSuite) TestSendReceiveParallel() {
	if testing.Short() {
		suite.T().Skip("Skipping TestSendReceiveParallel in short mode")
	}

	workers := 200
	inbound := make(chan message.Request, workers)
	trans := suite.Transport
	trans.Listen(testService, inbound)

	// Receive requests and respond to them all (with an identical payload)
	go func() {
		for {
			select {
			case <-trans.Tomb().Dying():
				return
			case req := <-inbound:
				rsp := message.NewResponse()
				rsp.SetId(req.Id())
				rsp.SetService(req.Service())
				rsp.SetEndpoint(req.Endpoint())
				rsp.SetPayload(req.Payload())
				suite.Assert().NoError(trans.Respond(req, rsp))
			}
		}
	}()

	wg := sync.WaitGroup{}
	wg.Add(workers)
	work := func(i int) {
		defer wg.Done()
		time.Sleep(time.Duration(rand.Float64()*250) * time.Millisecond)
		req := message.NewRequest()
		req.SetId(strconv.Itoa(i))
		req.SetService(testService)
		req.SetEndpoint("foo")
		req.SetPayload([]byte(strconv.Itoa(i)))
		rsp, err := trans.Send(req, time.Second)
		suite.Assert().NoError(err)
		suite.Assert().NotNil(rsp)
		suite.Assert().Equal(string(req.Payload()), string(rsp.Payload()))
	}
	for i := 0; i < workers; i++ {
		go work(i)
	}
	wg.Wait()
}
