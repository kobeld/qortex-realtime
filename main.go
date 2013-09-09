package main

import (
	"code.google.com/p/go.net/websocket"
	"github.com/kobeld/qortex-realtime/configs"
	"github.com/kobeld/qortex-realtime/handlers"
	"log"
	"net/http"
)

func main() {

	http.Handle("/conn", websocket.Handler(handlers.BuildConnection))

	log.Printf("Starting websocket server on %s\n", configs.WSPort)
	err := http.ListenAndServe(configs.WSPort, nil)
	if err != nil {
		panic("ListenAndServe: " + err.Error())
	}
}
