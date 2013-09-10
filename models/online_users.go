package models

import (
	"code.google.com/p/go.net/websocket"
	"github.com/theplant/qortex/users"
	"github.com/theplant/qortex/utils"
	"log"
	"sync"
	"time"
)

type OnlineUser struct {
	InActivedOrg  *ActiveOrg
	WsConns       []*websocket.Conn
	User          *users.User
	NewMessageIds []string
	Send          chan GenericPushingMessage
	Lock          sync.Mutex
	CloseTimer    *time.Timer
}

// Push realtime message from server to client
func (this *OnlineUser) PushToClient() {
	for ntf := range this.Send {
		for _, ws := range this.WsConns {
			err := websocket.JSON.Send(ws, ntf)
			if err != nil {
				log.Printf("WS %+v: Send %+v to %+v error! \n", ws, ntf, this.User.Email)
			}
		}
	}
}

func (this *OnlineUser) SendReply(reply GenericPushingMessage) {
	defer func() {
		if err := recover(); err != nil {
			utils.PrintStackAndError(err.(error))
		}
	}()
	this.Send <- reply
}
