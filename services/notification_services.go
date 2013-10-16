package services

import (
	"github.com/kobeld/qortex-realtime/global"
	"github.com/kobeld/qortex-realtime/models"
	"github.com/kobeld/qortex-realtime/models/counts"
	"github.com/kobeld/qortex-realtime/models/notifications"
	"github.com/kobeld/qortex-realtime/models/ws"
	"github.com/sunfmin/mgodb"
	"github.com/theplant/qortex/nsqproducers"
	"github.com/theplant/qortex/utils"
	"github.com/theplant/qortexapi"
	"labix.org/v2/mgo/bson"
	"sort"
)

func SendEntryNotification(entryTopicData *nsqproducers.EntryTopicData) (err error) {

	activeOrg, onlineUser, err := getActiveOrgAndOnlineUser(entryTopicData.OrgId, entryTopicData.UserId)
	if err != nil {
		utils.PrintStackAndError(err)
		return
	}
	apiEntry := entryTopicData.ApiEntry

	gdb, err := GetGroupOrgDB(entryTopicData.OrgId, apiEntry.GroupId)
	if err != nil {
		utils.PrintStackAndError(err)
		return
	}

	currentOrg := activeOrg.Organization
	currentUser := onlineUser.User
	groupId := bson.ObjectIdHex(apiEntry.GroupId)

	// Get the abstract Entity
	var entity Entity

	switch entryTopicData.Status {
	case nsqproducers.TOPIC_STATUS_LIKE, nsqproducers.TOPIC_STATUS_REMOVE_LIKE:
		hasLiked := entryTopicData.Status == nsqproducers.TOPIC_STATUS_LIKE
		entity = NewLikeEntity(gdb, currentOrg, currentUser, apiEntry, hasLiked)

	case nsqproducers.TOPIC_STATUS_CREATE:

		switch {
		case apiEntry.IsQortexSupport:
			entity = NewQortexSupportEntity(gdb, currentOrg, currentUser, apiEntry)

		default:
			entity = NewEntryEntity(gdb, currentOrg, currentUser, apiEntry)
		}
	}

	events, err := entity.MakeEventsAndSaveNotifications()
	if err != nil {
		utils.PrintStackAndError(err)
		return
	}

	var userOrgId bson.ObjectId
	onlineUsers := GetOnlineUsersByOrgIds(entity.GetToNotifyOrgIds())

	for _, event := range events {
		userOrgId = bson.ObjectIdHex(event.ToUser.OriginalOrgId)

		counts.ResetCount(gdb, userOrgId, event.ToUser.Id, groupId)

		onlineUser := pickOnlineUser(event.ToUser.Id, onlineUsers)

		if onlineUser == nil {
			continue
		}

		totolCount := counts.SumAndGetAllDbCount(onlineUser.AllDBs(), userOrgId, event.ToUser.Id)

		reply := CountNotification{
			Method:  "Counter.Refresh",
			GroupId: apiEntry.GroupId,
			MyCount: totolCount.ToApiCount(),
		}

		if event.ShowNewBar {
			reply.Method = "Counter.NewArrived"
			reply.NewEntry = true
			reply.EntryId = apiEntry.Id
			reply.NewMessageNumber = onlineUser.AddNewMessageId(apiEntry.Id)
		}

		onlineUser.SendReply(reply)
	}

	return
}

// TODO: should refactor this when the Shared ActiveOrg is implmented by Redis or others
func GetNotifications(allDbs []*mgodb.Database, orgId, userId bson.ObjectId, before string,
	limit int) (notifItems []*qortexapi.NotificationItem, err error) {

	beforeTime := global.ConvertStrToTime(before)
	notifisChan := make(chan []*notifications.Notification)

	for _, db := range allDbs {
		go notifications.PutNotificationsIntoChan(db, orgId, userId, beforeTime, limit, notifisChan)
	}

	notifis := notifications.GetNotificationsFromChan(notifisChan, len(allDbs))

	sort.Sort(ByNewAndNotifiedAt{notifis})
	if len(notifis) > limit {
		notifis = notifis[0:limit]
	}

	notifItems = models.ToApiNotificationItems(notifis)

	return
}

// ------ Private Methods -----
func pickOnlineUser(userId bson.ObjectId, onlineUsers map[bson.ObjectId]*ws.OnlineUser) (onlineUser *ws.OnlineUser) {
	for _, ou := range onlineUsers {
		if ou.User.Id == userId {
			onlineUser = ou
			break
		}
	}

	return
}
