package services

import (
	"github.com/kobeld/qortex-realtime/models/notifications"
	"github.com/sunfmin/mgodb"
	"github.com/theplant/qortex/entries"
	"github.com/theplant/qortex/groups"
	"github.com/theplant/qortex/organizations"
	"github.com/theplant/qortex/sharings"
	"github.com/theplant/qortex/users"
	"github.com/theplant/qortex/utils"
	"github.com/theplant/qortexapi"
	"labix.org/v2/mgo/bson"
	"time"
)

type Entity interface {
	// User who made this notification
	CausedByUser() *users.User
	CausedByOrg() *organizations.Organization
	// For events for every related user
	MakeEventsAndSaveNotifications() ([]*notifications.Event, error)

	GetToNotifyOrgIds() []string
}

// Generic struct implementing the default interface method
type baseEntity struct {
	gdb      *mgodb.Database // For handling shared group related events
	org      *organizations.Organization
	user     *users.User
	apiEntry *qortexapi.Entry
}

func (this *baseEntity) CausedByUser() *users.User {
	return this.user
}

func (this *baseEntity) CausedByOrg() *organizations.Organization {
	return this.org
}

func (this *baseEntity) GetToNotifyOrgIds() (orgIds []string) {
	orgIds = []string{this.org.Id.Hex()}

	groupId, err := utils.ToObjectId(this.apiEntry.GroupId)
	if err != nil {
		utils.PrintStackAndError(err)
		return
	}

	hostOrgId, accpetedOrgIds := sharings.GetAcceptedOrgObjectIds(groupId)
	if hostOrgId != "" {
		orgIds = utils.TurnObjectIdToPlainIds(append(accpetedOrgIds, hostOrgId))
	}

	return
}

// ----- Entry entity
type EntryEntity struct {
	baseEntity
}

func NewEntryEntity(gdb *mgodb.Database, org *organizations.Organization, user *users.User,
	apiEntry *qortexapi.Entry) (entryEntity *EntryEntity) {

	entryEntity = new(EntryEntity)
	entryEntity.gdb = gdb
	entryEntity.org = org
	entryEntity.user = user
	entryEntity.apiEntry = apiEntry

	return
}

func (this *EntryEntity) MakeEventsAndSaveNotifications() (events []*notifications.Event, err error) {
	groupId, err := utils.ToObjectId(this.apiEntry.GroupId)
	if err != nil {
		utils.PrintStackAndError(err)
		return
	}

	// Special cases:
	// 1. knowledge base: should always create notifications for all group followers
	// 2. Mentioned or notified users: should create notifications and show the new message bar,
	//    no matter they are following the group or not.

	// Merge notified and mentioned user for creating notification
	toUsersMap := make(map[string]string)
	for _, mentioned := range this.apiEntry.MentionedUsers {
		toUsersMap[mentioned.Id] = mentioned.Id
	}
	for _, notified := range this.apiEntry.ToUsers {
		if _, ok := toUsersMap[notified.Id]; !ok {
			toUsersMap[notified.Id] = notified.Id
		}
	}

	myGroups, err := groups.FindAllMyGroupsByGroupId(this.gdb, groupId)
	if err != nil {
		utils.PrintStackAndError(err)
		return
	}

	myGroupsMap := make(map[bson.ObjectId]*groups.MyGroup)
	for _, myGroup := range myGroups {
		myGroupsMap[myGroup.UserId] = myGroup
	}

	allUsers, err := users.FindAll(this.gdb, nil)
	if err != nil {
		utils.PrintStackAndError(err)
		return
	}

	fromUser := this.CausedByUser().ToEmbedUser()
	eventType := notifications.DecideEventType(this.apiEntry)

	createdAt := time.Now()
	allNotifis := []*notifications.Notification{}

	for _, user := range allUsers {

		// Don't notify the author
		if user.Id == fromUser.Id {
			continue
		}

		toUser := user.ToEmbedUser()

		// Show the "{n} New Messages" bar for group followers
		showNewBar := false
		if myGroup, ok := myGroupsMap[user.Id]; ok {
			showNewBar = myGroup.IsFollower
		}

		// Make event for all users
		event := notifications.NewEvent(&toUser, eventType, showNewBar)

		switch eventType {
		case notifications.VT_NEW_POST, notifications.VT_NEW_COMMENT:
			if _, ok := toUsersMap[user.Id.Hex()]; ok {
				event.ShowNewBar = true
				// Creating notification item
				event.Notification = notifications.NewNotification(&toUser, &fromUser,
					eventType, this.apiEntry, createdAt)
				allNotifis = append(allNotifis, event.Notification)

			}
		case notifications.VT_NEW_KNOWLEDGE:
			event.Notification = notifications.NewNotification(&toUser, &fromUser,
				eventType, this.apiEntry, createdAt)
			allNotifis = append(allNotifis, event.Notification)
		}

		events = append(events, event)
	}

	if len(allNotifis) == 0 {
		return
	}

	if err = notifications.SaveNotifications(this.gdb, allNotifis); err != nil {
		utils.PrintStackAndError(err)
		return
	}

	return
}

// ----- Like entity -----
type LikeEntity struct {
	baseEntity
	hasLiked bool
}

func NewLikeEntity(gdb *mgodb.Database, org *organizations.Organization, user *users.User,
	apiEntry *qortexapi.Entry, hasLiked bool) (likeEntity *LikeEntity) {

	likeEntity = new(LikeEntity)
	likeEntity.gdb = gdb
	likeEntity.org = org
	likeEntity.user = user
	likeEntity.apiEntry = apiEntry
	likeEntity.hasLiked = hasLiked

	return
}

func (this *LikeEntity) MakeEventsAndSaveNotifications() (events []*notifications.Event, err error) {

	authorId := bson.ObjectIdHex(this.apiEntry.AuthorId)

	// Don't notify yourself when you are narcissistic
	if this.CausedByUser().Id == authorId {
		return
	}

	// Find the Author who will be notified
	author, err := users.FindById(this.gdb, authorId)
	if err != nil {
		utils.PrintStackAndError(err)
		return
	}

	fromUser := this.user.ToEmbedUser()
	toUser := author.ToEmbedUser()

	if this.apiEntry.IsComment {
		rootEntry, err := entries.FindById(this.gdb, bson.ObjectIdHex(this.apiEntry.RootId))
		if err != nil {
			utils.PrintStackAndError(err)
			return events, err
		}
		this.apiEntry.Title = rootEntry.Title
	}

	eventType := notifications.VT_LIKE
	if !this.hasLiked {
		eventType = notifications.VT_REMOVE_LIKE
	}

	// If liking a comment, then should get the root entry title as the notificaiton title
	event := notifications.NewEvent(&toUser, eventType, false)
	event.Notification = notifications.NewNotification(&toUser, &fromUser, eventType, this.apiEntry, time.Now())
	events = append(events, event)

	if this.hasLiked {
		// Save notificaiton
		err = event.Notification.Save(this.gdb)
	} else {
		// Remove notification
		query := bson.M{
			"etype":       notifications.VT_LIKE,
			"userid":      toUser.Id,
			"entryid":     event.Notification.EntryId,
			"fromuser.id": fromUser.Id,
		}

		_, err = notifications.DeleteNotification(this.gdb, query)
	}

	if err != nil {
		utils.PrintStackAndError(err)
		return
	}

	return

}

// ----- Qortex Support Entity -----
type QortexSupportEntity struct {
	baseEntity
	toNotifyOrgIds []string
}

func NewQortexSupportEntity(gdb *mgodb.Database, org *organizations.Organization, user *users.User,
	apiEntry *qortexapi.Entry) (qtEntity *QortexSupportEntity) {

	qtEntity = new(QortexSupportEntity)
	qtEntity.gdb = gdb
	qtEntity.org = org
	qtEntity.user = user
	qtEntity.apiEntry = apiEntry
	qtEntity.toNotifyOrgIds = []string{}
	return
}

func (this *QortexSupportEntity) MakeEventsAndSaveNotifications() (events []*notifications.Event, err error) {

	entryId := bson.ObjectIdHex(this.apiEntry.Id)

	entry, err := entries.FindById(this.gdb, entryId)
	if err != nil {
		utils.PrintStackAndError(err)
		return
	}

	// Get all users that should be notified
	notifiedUsers, orgs, err := entry.GetQortexSupportAudiencesAndOrgs()
	if err != nil {
		utils.PrintStackAndError(err)
		return
	}

	// Set the apiEntry Title with the Root Entry Title, for the use of Notification Title
	// The CachedRootEntry is filled by GetQortexSupportAudiencesAndOrgs() above
	this.apiEntry.Title = entry.Transients.CachedRootEntry.Title

	for _, org := range orgs {
		this.toNotifyOrgIds = append(this.toNotifyOrgIds, org.Id.Hex())
	}

	// Get the event type
	var eventType string
	if this.apiEntry.IsComment {
		if this.apiEntry.IsFeedback {
			eventType = notifications.VT_NEW_QORTEX_FEEDBACK_COMMENT
		} else {
			eventType = notifications.VT_NEW_QORTEX_BROADCAST_COMMENT
		}
	} else {
		if this.apiEntry.IsFeedback {
			eventType = notifications.VT_NEW_QORTEX_FEEDBACK
		} else {
			eventType = notifications.VT_NEW_QORTEX_BROADCAST
		}
	}

	createdAt := time.Now()
	fromUser := this.CausedByUser().ToEmbedUser()
	allNotifis := []*notifications.Notification{}
	// var toUser users.EmbedUser

	for _, user := range notifiedUsers {

		if user.Id == fromUser.Id {
			continue
		}

		toUser := user.ToEmbedUser()
		event := notifications.NewEvent(&toUser, eventType, true)
		event.Notification = notifications.NewNotification(&toUser, &fromUser, eventType,
			this.apiEntry, createdAt)

		allNotifis = append(allNotifis, event.Notification)
		events = append(events, event)
	}

	if len(allNotifis) == 0 {
		return
	}

	if err = notifications.SaveNotifications(this.gdb, allNotifis); err != nil {
		utils.PrintStackAndError(err)
		return
	}

	return

}

func (this *QortexSupportEntity) GetToNotifyOrgIds() []string {
	return this.toNotifyOrgIds
}

// ----- Share Request Entity -----

type ShareRequestEntity struct {
	baseEntity
	apiRequest   *qortexapi.ShareRequest
	requestEntry *entries.Entry
}

func NewShareRequestEntity(org *organizations.Organization, user *users.User,
	apiRequest *qortexapi.ShareRequest) (requestEntity *ShareRequestEntity, err error) {

	requestEntity = new(ShareRequestEntity)
	requestEntity.user = user
	requestEntity.org = org
	requestEntity.apiRequest = apiRequest
	requestEntity.requestEntry, err = entries.FindShareRequestEntryById(org.Database, bson.ObjectIdHex(apiRequest.Id))
	if err != nil {
		utils.PrintStackAndError(err)
		return
	}

	return
}

func (this *ShareRequestEntity) GetRequestEntry() *entries.Entry {
	return this.requestEntry
}

func (this *ShareRequestEntity) GetApiShareRequest() *qortexapi.ShareRequest {
	return this.apiRequest
}

func (this *ShareRequestEntity) GetToNotifyOrgIds() []string {
	return []string{this.org.Id.Hex()}
}

func (this *ShareRequestEntity) MakeEventsAndSaveNotifications() (events []*notifications.Event, err error) {

	// Check if going to send realtime notification to Inviter organization users,
	// the behavior should be different to Inviter and Invitee org.
	isToInviterOrg := false
	if this.CausedByUser().Id.Hex() == this.apiRequest.FromUser.Id {
		isToInviterOrg = true
	}

	db := this.org.Database
	toUsers := []*users.User{}

	var eventType string
	switch {
	case this.apiRequest.IsPending:
		// Notify all except the Inviter
		toUsers, _ = users.FindAllExcept(db, []bson.ObjectId{this.user.Id})
		eventType = notifications.VT_NEW_SHARED_REQUEST

	case this.apiRequest.IsForwarded:

		eventType = notifications.VT_FORWARDED_SHARED_REQUEST
		if isToInviterOrg {
			// Only notify the Inviter
			toUsers = append(toUsers, this.user)
		} else {
			// Only notify super users
			toUsers, _ = users.FindSuperUsers(db)
		}

	case this.apiRequest.IsAccepted, this.apiRequest.IsRejected:

		if this.apiRequest.IsAccepted {
			eventType = notifications.VT_ACCEPT_SHARED_REQUEST
		} else {
			eventType = notifications.VT_REJECT_SHARED_REQUEST
			if this.apiRequest.ToOrg.Id == "" {
				eventType = notifications.VT_REJECT_BEFORE_FORWARDING
			}
		}

		if isToInviterOrg {
			// Only notify the Inviter
			toUsers = append(toUsers, this.user)
		} else {
			// The request has been forwarded and approved by admins,
			// then should notify the forwarder (real invitee).
			if this.apiRequest.Responser.Id != this.apiRequest.ToUser.Id {
				toUser, _ := users.FindById(db, bson.ObjectIdHex(this.apiRequest.ToUser.Id))
				if toUser != nil {
					toUsers = append(toUsers, toUser)
				}
			}
		}

	case this.apiRequest.IsCanceled:
		eventType = notifications.VT_CANCEL_SHARED_REQUEST
		toUsers = append(toUsers, this.user)

	case this.apiRequest.IsStopped:
		eventType = notifications.VT_STOP_SHARING_GROUP
		toUsers = append(toUsers, this.user)
	}

	toUsersMap := make(map[bson.ObjectId]bson.ObjectId)
	for _, toUser := range toUsers {
		toUsersMap[toUser.Id] = toUser.Id
	}

	fromUser := this.CausedByUser().ToEmbedUser()
	createdAt := time.Now()
	allNotifis := []*notifications.Notification{}

	// Initialize events for all users
	allUsers, _ := users.FindAll(db, nil)

	for _, user := range allUsers {
		// if user.Id == fromUser.Id {
		// 	continue
		// }

		toUser := user.ToEmbedUser()
		// Make event for user
		event := notifications.NewEvent(&toUser, eventType, false)
		if _, ok := toUsersMap[user.Id]; ok {
			// Creating notification item
			event.Notification = &notifications.Notification{
				UserId:         toUser.Id,
				OrgId:          this.org.Id,
				FromUser:       &fromUser,
				EntryId:        this.requestEntry.Id,
				GroupId:        this.requestEntry.GroupId,
				Title:          this.requestEntry.Title,
				Content:        this.requestEntry.Content,
				EType:          eventType,
				CreatedAt:      createdAt,
				RootId:         this.requestEntry.Id,
				RequestToEmail: this.apiRequest.ToEmail,
			}
			allNotifis = append(allNotifis, event.Notification)

		}

		events = append(events, event)
	}

	if len(allNotifis) == 0 {
		return
	}

	if err = notifications.SaveNotifications(db, allNotifis); err != nil {
		utils.PrintStackAndError(err)
		return
	}

	return
}
