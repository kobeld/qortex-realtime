package handlers

import (
	"code.google.com/p/go.net/websocket"
	"net/rpc/jsonrpc"
)

func BuildConnection(conn *websocket.Conn) {

	println("The connection is built")

	jsonrpc.ServeConn(conn)
}
