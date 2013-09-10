package main

import (
	"code.google.com/p/go.net/websocket"
	"github.com/kobeld/qortex-realtime/configs"
	"github.com/kobeld/qortex-realtime/services"
	"log"
	"net/http"
)

func main() {

	http.Handle("/conn", websocket.Handler(services.BuildConnection))

	log.Printf("Starting websocket server on %s\n", configs.WSPort)
	err := http.ListenAndServe(configs.WSPort, nil)
	if err != nil {
		panic("ListenAndServe: " + err.Error())
	}
}
