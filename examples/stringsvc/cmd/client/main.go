package main

import (
	"github.com/monzo/typhon"
	"github.com/monzo/typhon/examples/stringsvc/pkg/stringsvc"
	"log"
)

func main() {
	client := stringsvc.NewClient("http://localhost:8085", typhon.Client.Filter(typhon.ErrorFilter).Filter(typhon.H2cFilter))

	resp, err := client.Uppercase("Hello world")
	if err != nil {
		panic(err)
	}
	log.Printf("Got uppercase response: %s", resp)

	count, err := client.Count(resp)
	if err != nil {
		panic(err)
	}
	log.Printf("Got count response: %d", count)
}
