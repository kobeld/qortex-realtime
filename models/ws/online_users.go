package ws

import (
	"code.google.com/p/go.net/websocket"
	"github.com/kobeld/qortex-realtime/configs"
	"github.com/sunfmin/mgodb"
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

func (this *OnlineUser) AllDBs() []*mgodb.Database {
	return this.InActivedOrg.AllDBs
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

func (this *OnlineUser) ClearNewMessageId() int {
	this.Lock.Lock()
	defer this.Lock.Unlock()
	this.NewMessageIds = []string{}

	return 0
}

func (this *OnlineUser) DeleteNewMessageId(entryId string) int {
	this.Lock.Lock()
	defer this.Lock.Unlock()

	for i, id := range this.NewMessageIds {
		if id == entryId {
			this.NewMessageIds = append(this.NewMessageIds[:i], this.NewMessageIds[i+1:]...)
		}
	}
	return len(this.NewMessageIds)
}

func (this *OnlineUser) AddNewMessageId(entryId string) int {
	this.Lock.Lock()
	defer this.Lock.Unlock()

	// No duplicated id
	for _, id := range this.NewMessageIds {
		if id == entryId {
			return len(this.NewMessageIds)
		}
	}

	this.NewMessageIds = append(this.NewMessageIds, entryId)
	return len(this.NewMessageIds)
}

func (this *OnlineUser) KillWebsocket(conn *websocket.Conn) {
	this.Lock.Lock()
	defer this.Lock.Unlock()

	for index, wsConn := range this.WsConns {
		if wsConn == conn {
			this.WsConns = append(this.WsConns[:index], this.WsConns[index+1:]...)
			log.Printf("Killing WebSocket for %+v, left %+v connection.  \n", this.User.Email, len(this.WsConns))
		}
	}
	conn.Close()

	if len(this.WsConns) == 0 {
		if this.CloseTimer != nil {
			this.CloseTimer.Stop()
		}
		this.CloseTimer = time.AfterFunc(configs.ONLINE_USER_CLOSE_DURATION, func() {
			log.Printf("Websocket: No other living BrowserSockets. Cleaning ( %+v ) resources. \n", this.User.Email)
			this.InActivedOrg.KillUser(this.User.Id.Hex())

			// Update user offline time and put user into the offline queue for getting offline digest mail
			this.User.UpdateOfflineTime(this.InActivedOrg.Organization.Database)

			// TODO: Enable it later
			// PutOfflineUserIntoQueue(this.User, this.InOrganization.OrganizationId)
		})
	}
}
