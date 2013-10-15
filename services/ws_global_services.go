package services

import (
	"github.com/kobeld/qortex-realtime/models/ws"
	"github.com/sunfmin/mgodb"
	"github.com/theplant/qortex/organizations"
	"github.com/theplant/qortex/utils"
	"labix.org/v2/mgo/bson"
	"sync"
)

var mu sync.Mutex

// The map key is OrganizationId
var activeOrgMap = make(map[string]*ws.ActiveOrg)

func MyActiveOrg(orgIdHex string) (activeOrg *ws.ActiveOrg, err error) {
	mu.Lock()
	defer mu.Unlock()

	// Validation: The org id should be valid
	orgId, err := utils.ToObjectId(orgIdHex)
	if err != nil {
		utils.PrintStackAndError(err)
		return
	}

	// Already running in the map
	activeOrg, exist := activeOrgMap[orgIdHex]
	if exist {
		return
	}

	// Should init the org and put into map for further use
	org, err := organizations.FindById(orgId)
	if err != nil {
		utils.PrintStackAndError(err)
		return
	}

	// Find and maintain all dbs for handling shared groups
	allDBs, err := org.GetCurrentAndEmbedDBs()
	if err != nil {
		utils.PrintStackAndError(err)
		return
	}

	// Init the activeOrg and put it into the map
	activeOrg = &ws.ActiveOrg{
		OrgId:        orgIdHex,
		Organization: org,
		OnlineUsers:  make(map[string]*ws.OnlineUser),
		Broadcast:    make(chan ws.GenericPushingMessage),
		CloseSign:    make(chan bool),
		AllDBs:       allDBs,
	}

	go runActiveOrg(activeOrg)
	activeOrgMap[orgIdHex] = activeOrg

	return
}

// The heart of ActiveOrg
func runActiveOrg(activeOrg *ws.ActiveOrg) {
	for {
		select {
		case b := <-activeOrg.Broadcast:
			for _, ou := range activeOrg.OnlineUsers {
				ou.Send <- b
			}
		case c := <-activeOrg.CloseSign:
			if c == true {
				delete(activeOrgMap, activeOrg.OrgId)
				close(activeOrg.Broadcast)
				close(activeOrg.CloseSign)
				return
			}
		}
	}
}

// Get the group located Organization by the group Id
func GetGroupOrg(orgIdHex, groupIdHex string) (org *organizations.Organization, err error) {
	activeOrg, err := MyActiveOrg(orgIdHex)
	if err != nil {
		utils.PrintStackAndError(err)
		return
	}

	groupOrgIdHex, exist := activeOrg.Organization.GroupToOrgMap[groupIdHex]
	if groupIdHex == "" || !exist {
		org = activeOrg.Organization
		return
	}

	// TODO: Could let ActiveOrg maintain a map[orgId]*org map that caches all GroupToOrgMap
	return organizations.FindById(bson.ObjectIdHex(groupOrgIdHex))
}

func GetGroupOrgDB(orgIdHex, groupIdHex string) (db *mgodb.Database, err error) {
	org, err := GetGroupOrg(orgIdHex, groupIdHex)
	if err != nil {
		utils.PrintStackAndError(err)
		return
	}

	db = org.Database
	return
}

func GetOnlineUsersByOrgIds(orgIds []string) map[bson.ObjectId]*ws.OnlineUser {
	onlineUsers := make(map[bson.ObjectId]*ws.OnlineUser)
	for _, orgId := range orgIds {
		org, _ := MyActiveOrg(orgId)
		if org == nil {
			continue
		}

		for _, onlineUser := range org.OnlineUsers {
			onlineUsers[onlineUser.User.Id] = onlineUser
		}
	}
	return onlineUsers
}

// Handy function for getting ActiveOrg and OnlineUser when passing orgId and userId in
func getActiveOrgAndOnlineUser(orgIdHex, userIdHex string) (activeOrg *ws.ActiveOrg, onlineUser *ws.OnlineUser, err error) {
	activeOrg, err = MyActiveOrg(orgIdHex)
	if err != nil {
		utils.PrintStackAndError(err)
		return
	}

	onlineUser = activeOrg.GetOnlineUserById(userIdHex)

	return
}
