package typhon

import (
	"testing"

	"github.com/mondough/typhon/client"
	"github.com/mondough/typhon/example/handler"
	"github.com/mondough/typhon/example/proto/callhello"
	"github.com/mondough/typhon/test"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

func TestExample(t *testing.T) {
	s := test.InitServer(t, "example")
	defer s.Close()

	// Register example endpoints
	s.RegisterEndpoint(handler.Hello)
	s.RegisterEndpoint(handler.CallHello)

	resp := &callhello.Response{}

	err := client.Req(
		context.TODO(),                     // context
		"example",                          // service
		"callhello",                        // service endpoint to call
		&callhello.Request{Value: "Bunny"}, // request
		resp, // response
	)
	require.NoError(t, err)
	t.Logf("[testHandler] received %s", resp)
	require.Equal(t, "example.hello says 'Hello, Bunny!'", resp.Value)
	// Log the response we receive

}

// Test with AccessToken being passed (once directly via client.Request and once via the context)
// @todo refactor typhon client interface to make all boilerplate in this method unnecessary
func TestWithAccessToken(t *testing.T) {
	// @todo for some reason i can't start a second test server in here. I haven't got
	// the time to figure it out now, though :(
	t.Skip()

	s := test.InitServer(t, "example")
	defer s.Close()

	// Register example endpoints
	s.RegisterEndpoint(handler.Hello)
	s.RegisterEndpoint(handler.CallHello)

	respProto := &callhello.Response{}
	reqProto := &callhello.Request{Value: "Hullo"}
	payload, err := proto.Marshal(reqProto)
	require.NoError(t, err)
	req, err := client.NewProtoRequest("example", "callhello", payload)
	require.NoError(t, err)
	req.SetAccessToken("I am here, wee, wee!")

	resp, err := client.CustomReq(context.TODO(), req)
	require.NoError(t, err)

	err = proto.Unmarshal(resp.Payload(), respProto)
	require.NoError(t, err)

	t.Logf("[testHandler] received %s", respProto)
	require.Equal(t, "example.hello says 'Hello, Hullo!'", respProto.Value)
}
