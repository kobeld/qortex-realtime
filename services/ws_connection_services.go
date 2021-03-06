package services

import (
	"code.google.com/p/go.net/websocket"
	"github.com/sunfmin/signature"
	"github.com/theplant/qortex/configs"
	"github.com/theplant/qortex/members"
	"github.com/theplant/qortex/users"
	"github.com/theplant/qortex/utils"
	"labix.org/v2/mgo/bson"
	"log"
	"net/rpc/jsonrpc"
	"runtime/debug"
)

// Entrance that builds and maintains the websocket connection for users
func BuildConnection(conn *websocket.Conn) {

	defer func() {
		if err := recover(); err != nil {
			log.Printf("********** WebSocket Error: %+v ***********\n", err)
			debug.PrintStack()
		}
	}()

	orgIdHex := conn.Request().URL.Query().Get("o")
	if orgIdHex == "" {
		return
	}

	wsCookie := ""
	for _, cc := range conn.Request().Cookies() {
		if cc.Name == "qortex" {
			wsCookie = cc.Value
			break
		}
	}
	if wsCookie == "" {
		return
	}

	member, _ := getSessionMember(wsCookie)
	if member == nil {
		return
	}

	activeOrg, err := MyActiveOrg(orgIdHex)
	if err != nil {
		utils.PrintStackAndError(err)
		return
	}

	user, err := users.FindById(activeOrg.Organization.Database, member.Id)
	if err != nil {
		utils.PrintStackAndError(err)
		return
	}

	onlineUser := activeOrg.GetOrInitOnlineUser(user, conn)
	log.Printf("----> New websocket connection for: %s, %+v running totally",
		user.Email, len(onlineUser.WsConns))

	// Holding the connection
	jsonrpc.ServeConn(conn)

	// Cut current connection and clean up related resources
	onlineUser.KillWebsocket(conn)
}

func getSessionMember(session string) (member *members.Member, err error) {
	var e map[string]interface{}
	if err = signature.DecodeString(session, &e, configs.SESSION_SECRET); err != nil {
		return
	}
	if _, ok := e["id"]; !ok {
		return
	}
	member, err = members.FindById(bson.ObjectIdHex(e["id"].(string)))
	if err != nil {
		return
	}

	return
}
