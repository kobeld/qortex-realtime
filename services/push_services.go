package services

import (
	"github.com/kobeld/qortex-realtime/models/counts"
	"github.com/kobeld/qortex-realtime/models/notifications"
	"github.com/kobeld/qortex-realtime/models/ws"
	"github.com/theplant/qortex/nsqproducers"
	"github.com/theplant/qortex/users"
	"github.com/theplant/qortex/utils"
	"github.com/theplant/qortex/wsdata"
	"labix.org/v2/mgo/bson"
)

func PushGroupNotification(groupTopicData *nsqproducers.GroupTopicData) (err error) {

	activeOrg, onlineUser, err := getActiveOrgAndOnlineUser(groupTopicData.OrgId, groupTopicData.UserId)
	if err != nil {
		utils.PrintStackAndError(err)
		return
	}

	var user *users.User
	if onlineUser == nil {
		user, err = users.FindById(activeOrg.Organization.Database, bson.ObjectIdHex(groupTopicData.UserId))
		if err != nil {
			utils.PrintStackAndError(err)
			return err
		}
	} else {
		user = onlineUser.User
	}

	entity, err := NewShareRequestEntity(activeOrg.Organization, user, groupTopicData.ApiRequest)
	if err != nil {
		utils.PrintStackAndError(err)
		return
	}

	events, err := entity.MakeEventsAndSaveNotifications()
	if err != nil {
		utils.PrintStackAndError(err)
		return
	}

	var userOrgId bson.ObjectId
	onlineUsers := GetOnlineUsersByOrgIds(entity.GetToNotifyOrgIds())
	requestEntryId := entity.GetRequestEntry().Id.Hex()
	groupId := entity.GetRequestEntry().GroupId

	for _, event := range events {

		userOrgId = bson.ObjectIdHex(event.ToUser.OriginalOrgId)

		counts.ResetCount(activeOrg.Organization.Database, userOrgId, event.ToUser.Id, groupId)

		toOnlineUser := pickOnlineUser(event.ToUser.Id, onlineUsers)

		if toOnlineUser == nil {
			continue
		}

		totolCount := counts.SumAndGetAllDbCount(toOnlineUser.AllDBs(), userOrgId, event.ToUser.Id)

		switch event.VType {
		case notifications.VT_NEW_SHARED_REQUEST:

			reply := CountNotification{
				Method:           "Counter.NewArrived",
				GroupId:          groupId.Hex(),
				MyCount:          totolCount.ToApiCount(),
				NewEntry:         true,
				EntryId:          requestEntryId,
				NewMessageNumber: toOnlineUser.AddNewMessageId(requestEntryId),
			}
			toOnlineUser.SendReply(reply)

		case notifications.VT_ACCEPT_SHARED_REQUEST, notifications.VT_REJECT_SHARED_REQUEST,
			notifications.VT_STOP_SHARING_GROUP, notifications.VT_CANCEL_SHARED_REQUEST,
			notifications.VT_REJECT_BEFORE_FORWARDING, notifications.VT_FORWARDED_SHARED_REQUEST:

			barHTML := groupTopicData.ApiRequest.RequestBarHtml

			// TODO: should move into this project
			reply := wsdata.TaskState{
				Method:  "Task.ChangeState",
				TaskBar: barHTML,
				EntryId: requestEntryId,
				MyCount: totolCount.ToApiCount(),
			}
			toOnlineUser.SendReply(reply)
		}
	}

	return
}

func PushEntryNotification(entryTopicData *nsqproducers.EntryTopicData) (err error) {

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

		toOnlineUser := pickOnlineUser(event.ToUser.Id, onlineUsers)

		if toOnlineUser == nil {
			continue
		}

		totolCount := counts.SumAndGetAllDbCount(toOnlineUser.AllDBs(), userOrgId, event.ToUser.Id)

		reply := CountNotification{
			Method:  "Counter.Refresh",
			GroupId: apiEntry.GroupId,
			MyCount: totolCount.ToApiCount(),
		}

		if event.ShowNewBar {
			reply.Method = "Counter.NewArrived"
			reply.NewEntry = true
			reply.EntryId = apiEntry.Id
			reply.NewMessageNumber = toOnlineUser.AddNewMessageId(apiEntry.Id)
		}

		toOnlineUser.SendReply(reply)
	}

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
