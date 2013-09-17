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
	allDBs := []*mgodb.Database{org.Database}
	embedOrgs, err := organizations.FindByIds(org.EmbededOrgIds)
	if err != nil {
		utils.PrintStackAndError(err)
		return
	}

	for _, embedOrg := range embedOrgs {
		allDBs = append(allDBs, embedOrg.Database)
	}

	// Init the activeOrg and put it into the map
	activeOrg = &ws.ActiveOrg{
		OrgId:        orgIdHex,
		Organization: org,
		OnlineUsers:  make(map[bson.ObjectId]*ws.OnlineUser),
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
