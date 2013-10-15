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
	// ApiEntry which triggered this notification
	CausedByEntry() *qortexapi.Entry
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

func (this *baseEntity) CausedByEntry() *qortexapi.Entry {
	return this.apiEntry
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

	// // Special cases:
	// // 1. knowledge base: should always create notifications for all group followers
	// // 2. Mentioned or notified users: should create notifications and show the new message bar,
	// //    no matter they are following the group or not.

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
