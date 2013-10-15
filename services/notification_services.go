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
	entity := NewEntryEntity(gdb, currentOrg, currentUser, apiEntry)

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

// ----- old entry
// func SendEntryNotification_old(entryTopicData *nsqproducers.EntryTopicData) (err error) {

// 	serv, err := MakeWsService(entryTopicData.OrgId, entryTopicData.UserId)
// 	if err != nil {
// 		utils.PrintStackAndError(err)
// 		return
// 	}

// 	apiEntry := entryTopicData.ApiEntry
// 	currentOrg := serv.CurrentOrg
// 	currentUser := serv.LoggedInUser

// 	// TODO: the db should be group db, not current org db
// 	db := currentOrg.Database

// 	entry, err := entries.FindById(db, bson.ObjectIdHex(apiEntry.Id))
// 	if err != nil {
// 		utils.PrintStackAndError(err)
// 		return
// 	}

// 	var entity notifications.Entity

// 	switch entryTopicData.Status {
// 	case nsqproducers.TOPIC_STATUS_CREATE, nsqproducers.TOPIC_STATUS_DELETE, nsqproducers.TOPIC_STATUS_UPDATE:
// 		entity = notifications.MakeEntryEntity(db, currentOrg, currentUser, entry)

// 	case nsqproducers.TOPIC_STATUS_LIKE, nsqproducers.TOPIC_STATUS_REMOVE_LIKE:
// 		hasLiked := (entryTopicData.Status == nsqproducers.TOPIC_STATUS_LIKE)
// 		entity = notifications.NewLikeEntity(currentOrg, currentUser, entry, hasLiked)
// 	}

// 	// currentTime := time.Now()
// 	causedEntry := entity.CausedEntry()
// 	causedEntries := entity.CausedEntries()
// 	eventMap := entity.Events(db)

// 	// Save or delete notification items
// 	if err := entity.HandleNotificationItems(db, eventMap); err != nil {
// 		utils.PrintStackAndError(err) // dont' return
// 	}

// 	orgIds := entity.GetToNotifyOrgIds()

// 	// Build organization map for shared group user
// 	orgMap := make(map[string]*organizations.Organization)
// 	orgs, err := organizations.FindByIds(utils.TurnPlainIdsToObjectIds(orgIds))
// 	if err == nil {
// 		for _, org := range orgs {
// 			orgMap[org.Id.Hex()] = org
// 		}
// 	}

// 	onlineUsers := GetOnlineUsersByOrgIds(orgIds)
// 	emailToUserMap := make(map[string]bool)

// 	// Handle event for each user
// 	for toUserId, event := range eventMap {
// 		// If it is Qortex Support, then the key of the eventMap is "userId-organizationId",
// 		// which is used to differentiate same user in different organizations
// 		toUserId = strings.Split(toUserId, "-")[0]

// 		toUserObjectId := bson.ObjectIdHex(toUserId)
// 		orgId := bson.ObjectIdHex(event.ToUser.OriginalOrgId)

// 		// don't notify anything to sender
// 		if currentUser.Id == toUserObjectId && entity.NotNotifySelf() {
// 			continue
// 		}

// 		if entity.NeedResetUserCount() {

// 			if causedEntries != nil {
// 				for _, causedEntry := range causedEntries {
// 					notifications.ResetCount(db, toUserObjectId, causedEntry.GroupId, orgId)
// 				}

// 			} else {
// 				// Reset user mycount for event user
// 				notifications.ResetCount(db, toUserObjectId, causedEntry.GroupId, orgId)
// 			}
// 		}

// 		onlineUser := pickOnlineUser(toUserObjectId, onlineUsers)
// 		if onlineUser == nil {

// 			// Don't send mail multi times to the same member when posting a Qortex Support
// 			_, exist := emailToUserMap[toUserId]
// 			if !exist && event.NeedToSendNotificationMail() {
// 				// Send Mail if user offline, only work in Dev and Production
// 				// serv.SendNotificationMail(event, apiEntry, currentTime, orgMap)
// 			}

// 		} else if entity.NeetToSendRealtimeNotification(onlineUser.User) {
// 			makeAndPushEventReply(currentUser, event, entity, onlineUser)
// 		}

// 		emailToUserMap[toUserId] = true
// 	}

// 	return
// }

// func makeAndPushEventReply(currentUser *users.User, event *notifications.Event,
// 	entity notifications.Entity, onlineUser *ws.OnlineUser) {

// 	// entry := entity.CausedEntry()

// 	switch event.VType {
// 	case notifications.VT_DEFAULT, notifications.VT_NEW_POST, notifications.VT_NEW_TODO,
// 		notifications.VT_NEW_CHAT, notifications.VT_POST_NEED_ACK, notifications.VT_COMMENT_NEED_ACK,
// 		notifications.VT_NEW_COMMENT, notifications.VT_NEW_KNOWLEDGE, notifications.VT_NEW_SHARED_REQUEST,
// 		notifications.VT_FORWARDED_SHARED_REQUEST, notifications.VT_NEW_QORTEX_BROADCAST,
// 		notifications.VT_NEW_QORTEX_FEEDBACK, notifications.VT_NEW_INNER_MESSAGE:

// 		reply := CountNotification{
// 			Method:  "Counter.Refresh",
// 			GroupId: entity.CausedEntry().GroupId.Hex(),
// 			MyCount: services.UserCountData(onlineUser.AllDBs(), onlineUser.User),
// 		}

// 		if event.IsFollowed && currentUser.Id != onlineUser.User.Id {
// 			reply.Method = "Counter.NewArrived"
// 			reply.NewEntry = true
// 			reply.EntryId = entity.NewEntryId().Hex()
// 			reply.NewMessageNumber = onlineUser.AddNewMessageId(reply.EntryId)
// 		}
// 		onlineUser.SendReply(reply)

// 	case notifications.VT_LIKE, notifications.VT_REMOVE_LIKE:

// 		reply := CountNotification{
// 			Method:  "Counter.Refresh",
// 			GroupId: entity.CausedEntry().GroupId.Hex(),
// 			MyCount: services.UserCountData(onlineUser.AllDBs(), onlineUser.User),
// 		}
// 		onlineUser.SendReply(reply)
// 	}
// 	return
// }
