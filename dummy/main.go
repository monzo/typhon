package main

import (
	"encoding/json"
	"fmt"

	"github.com/vinceprignano/bunny/server"
	"github.com/vinceprignano/bunny/transport/rabbit"
	"github.com/vinceprignano/bunny/transport/rabbit/endpoint"
)

func HelloHandler(req *rabbit.RabbitRequest) ([]byte, error) {
	reqBody := make(map[string]interface{})
	json.Unmarshal(req.Body(), reqBody)
	fmt.Println(reqBody)
	return json.Marshal(map[string]interface{}{
		"value": fmt.Sprintf("Hello, %s!", reqBody["value"]),
	})
}

func main() {
	bunnyServer := server.NewServer("helloworld", rabbit.NewRabbitTransport())
	bunnyServer.RegisterEndpoint(&endpoint.JsonEndpoint{
		EndpointName: "sayhello",
		Handler:      HelloHandler,
	})
	bunnyServer.Init()
}
