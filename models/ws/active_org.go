package ws

import (
	"code.google.com/p/go.net/websocket"
	"github.com/sunfmin/mgodb"
	"github.com/theplant/qortex/organizations"
	"github.com/theplant/qortex/users"
	"log"
	"sync"
)

// Should always contains a "Method" string for RPC protocal
type GenericPushingMessage interface{}

type ActiveOrg struct {
	OrgId        string
	Organization *organizations.Organization
	OnlineUsers  map[string]*OnlineUser
	Broadcast    chan GenericPushingMessage
	CloseSign    chan bool
	AllDBs       []*mgodb.Database
	Lock         sync.Mutex
}

func (this *ActiveOrg) GetOrInitOnlineUser(user *users.User, conn *websocket.Conn) (onlineUser *OnlineUser) {

	this.Lock.Lock()
	defer this.Lock.Unlock()

	onlineUser = this.OnlineUsers[user.Id.Hex()]
	if onlineUser == nil {
		log.Printf("----> New online user %s", user.Email)
		onlineUser = &OnlineUser{
			InActivedOrg: this,
			WsConns:      []*websocket.Conn{},
			User:         user,
			Send:         make(chan GenericPushingMessage, 32),
		}
		this.OnlineUsers[user.Id.Hex()] = onlineUser
		go onlineUser.PushToClient()
	}

	if onlineUser.CloseTimer != nil {
		onlineUser.CloseTimer.Stop()
	}
	onlineUser.WsConns = append(onlineUser.WsConns, conn)

	return
}

func (this *ActiveOrg) GetOnlineUserById(userIdHex string) (onlineUser *OnlineUser) {
	onlineUser, _ = this.OnlineUsers[userIdHex]
	return
}

func (this *ActiveOrg) KillUser(userIdHex string) {
	delete(this.OnlineUsers, userIdHex)
	// If no one in group, close and clean the resouce.
	if len(this.OnlineUsers) == 0 {
		this.CloseSign <- true
	}
}
