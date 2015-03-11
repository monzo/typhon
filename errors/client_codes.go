// This file maps error codes such as "service.ledger.invalid_amount" to integer
// codes that the iPhone app and other clients consume.
package errors

// The default error code is 1 (if no mapping was found)
const DEFAULT_CLIENT_CODE = 1

var ClientCodes = map[string]int{

	// 10xxx service.ledger
	"service.ledger.something": 10001,
}
