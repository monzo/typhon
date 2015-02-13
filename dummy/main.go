package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/vinceprignano/bunny"
	"github.com/vinceprignano/bunny/server"
	"github.com/vinceprignano/bunny/transport/rabbit"
	"github.com/vinceprignano/bunny/transport/rabbit/endpoint"
)

var bunnyServer *server.Server

func HelloHandler(req *rabbit.RabbitRequest) ([]byte, error) {
	reqBody := make(map[string]interface{})
	json.Unmarshal(req.Body(), &reqBody)
	return json.Marshal(map[string]interface{}{
		"Value": fmt.Sprintf("Hello, %s!", reqBody["Value"].(string)),
	})
}

func testBunny() {
	time.Sleep(1 * time.Second)
	body, _ := json.Marshal(map[string]interface{}{
		"Value": "Bunny",
	})
	bunnyServer.Transport.Publish("helloworld.sayhello", body)
}

func main() {
	bunnyServer = bunny.NewRabbitServer("helloworld")
	bunnyServer.RegisterEndpoint(&endpoint.JsonEndpoint{
		EndpointName: "sayhello",
		Handler:      HelloHandler,
	})
	bunnyServer.Init()
	go testBunny()
	bunnyServer.Run()
}
