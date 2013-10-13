package services

import (
	"github.com/kobeld/qortex-realtime/models/counts"
	"github.com/kobeld/qortex-realtime/models/notifications"
	"github.com/theplant/qortex/entries"
	"github.com/theplant/qortex/utils"
	"github.com/theplant/qortexapi"
	"labix.org/v2/mgo"
	"time"
)

func ReadEntry(orgIdHex, userIdHex, groupIdHex, entryIdHex string) (apiCount *qortexapi.MyCount, err error) {

	userId, err := utils.ToObjectId(userIdHex)
	if err != nil {
		utils.PrintStackAndError(err)
		return
	}

	entryId, err := utils.ToObjectId(entryIdHex)
	if err != nil {
		utils.PrintStackAndError(err)
		return
	}

	groupId, err := utils.ToObjectId(groupIdHex)
	if err != nil {
		utils.PrintStackAndError(err)
		return
	}

	gdb, err := GetGroupOrgDB(orgIdHex, groupIdHex)
	if err != nil {
		utils.PrintStackAndError(err)
		return
	}

	// Get the current active org
	activeOrg, err := MyActiveOrg(orgIdHex)
	if err != nil {
		utils.PrintStackAndError(err)
		return
	}

	orgId := activeOrg.Organization.Id

	// Handle EntriesReader
	err = entries.ReadEntryItem(gdb, userId, entryId)
	if err != nil && err != mgo.ErrNotFound {
		utils.PrintStackAndError(err)
		return
	}

	// Handle notificaiton
	err = notifications.ReadNotifications(gdb, userId, entryId, time.Now())
	if err != nil {
		utils.PrintStackAndError(err)
		return
	}

	// Reset user MyCount in current org
	_, err = counts.ResetCount(gdb, orgId, userId, groupId)
	if err != nil {
		utils.PrintStackAndError(err)
		return
	}

	apiCount = counts.SumAndGetAllDbCount(activeOrg.AllDBs, orgId, userId).ToApiCount()

	return
}
