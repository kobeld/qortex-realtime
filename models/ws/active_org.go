package ws

import (
	"code.google.com/p/go.net/websocket"
	"errors"
	"github.com/sunfmin/mgodb"
	"github.com/theplant/qortex/organizations"
	"github.com/theplant/qortex/users"
	"labix.org/v2/mgo/bson"
	"log"
	"sync"
)

// Should always contains a "Method" string for RPC protocal
type GenericPushingMessage interface{}

type ActiveOrg struct {
	OrgId        string
	Organization *organizations.Organization
	OnlineUsers  map[bson.ObjectId]*OnlineUser
	Broadcast    chan GenericPushingMessage
	CloseSign    chan bool
	AllDBs       []*mgodb.Database
	Lock         sync.Mutex
}

func (this *ActiveOrg) GetOrInitOnlineUser(user *users.User, conn *websocket.Conn) (onlineUser *OnlineUser) {

	this.Lock.Lock()
	defer this.Lock.Unlock()

	onlineUser = this.OnlineUsers[user.Id]
	if onlineUser == nil {
		log.Printf("----> New online user %s", user.Email)
		onlineUser = &OnlineUser{
			InActivedOrg: this,
			WsConns:      []*websocket.Conn{},
			User:         user,
			Send:         make(chan GenericPushingMessage, 32),
		}
		this.OnlineUsers[user.Id] = onlineUser
		go onlineUser.PushToClient()
	}

	if onlineUser.CloseTimer != nil {
		onlineUser.CloseTimer.Stop()
	}
	onlineUser.WsConns = append(onlineUser.WsConns, conn)

	return
}

func (this *ActiveOrg) GetOnlineUserById(userId bson.ObjectId) (onlineUser *OnlineUser, err error) {
	ok := false
	onlineUser, ok = this.OnlineUsers[userId]
	if !ok {
		err = errors.New("No such user in running Org")
		return
	}

	return
}

func (this *ActiveOrg) KillUser(userId bson.ObjectId) {
	delete(this.OnlineUsers, userId)
	// If no one in group, close and clean the resouce.
	if len(this.OnlineUsers) == 0 {
		this.CloseSign <- true
	}
}
