package rabbit

import (
	log "github.com/cihub/seelog"
	"github.com/streadway/amqp"
)

func headersToTable(headers map[string]string) amqp.Table {
	// Build an amqp.Table from the headers (how tedious)
	result := make(amqp.Table, len(headers))
	for k, v := range headers {
		result[k] = v
	}
	return result
}

func tableToHeaders(table amqp.Table) map[string]string {
	result := make(map[string]string, len(table))
	for k, v := range table {
		switch v := v.(type) {
		case string:
			result[k] = v
		default:
			log.Tracef("[Typhon:RabbitTransport] Received non-string header value for %s; discarding", k)
		}
	}
	return result
}
