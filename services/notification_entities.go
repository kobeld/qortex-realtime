package services

import (
	"github.com/kobeld/qortex-realtime/models/notifications"
	"github.com/sunfmin/mgodb"
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
	MakeEventsAndSaveNotifications() ([]*Event, error)

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

func (this *EntryEntity) MakeEventsAndSaveNotifications() (events []*Event, err error) {
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
	eType := decideEventType(this.apiEntry)

	createdAt := time.Now()
	allNotifis := []*notifications.Notification{}
	for _, user := range allUsers {

		// Don't notify the author
		if user.Id == fromUser.Id {
			continue
		}

		showNewBar := false
		if myGroup, ok := myGroupsMap[user.Id]; ok {
			showNewBar = myGroup.IsFollower
		}

		toUser := user.ToEmbedUser()
		event := NewEvent(&toUser, eType, showNewBar)
		if _, ok := toUsersMap[user.Id.Hex()]; ok {
			event.ShowNewBar = true
			// Creating notification item
			event.Notification = notifications.NewNotification(&toUser, &fromUser,
				eType, this.apiEntry, createdAt)
			allNotifis = append(allNotifis, event.Notification)

		}
		events = append(events, event)
	}

	return

	if err = notifications.SaveNotifications(this.gdb, allNotifis); err != nil {
		utils.PrintStackAndError(err)
		return
	}

	return
}
