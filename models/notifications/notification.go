package notifications

import (
	"fmt"
	"github.com/sunfmin/mgodb"
	"github.com/theplant/qortex/users"
	"github.com/theplant/qortex/utils"
	"github.com/theplant/qortexapi"
	"labix.org/v2/mgo/bson"
	"time"
)

const (
	CONTENT_STRING_LENGTH = 60
)

type Notification struct {
	Id        bson.ObjectId `bson:"_id"`
	UserId    bson.ObjectId
	OrgId     bson.ObjectId // Original Org Id
	GroupId   bson.ObjectId
	EntryId   bson.ObjectId // The entry caused this notification
	FromUser  *users.EmbedUser
	Title     string
	Content   string
	RootId    bson.ObjectId // Comment on Entry Id. When not a comment, it will be EntryId
	EType     string
	ReadAt    time.Time `bson:",omitempty"`
	CreatedAt time.Time
}

func (this *Notification) MakeId() interface{} {
	if this.Id == "" {
		this.Id = bson.NewObjectId()
	}
	return this.Id
}

func (this *Notification) Link() (url string) {
	baseUrl := fmt.Sprintf("#groups/%v", this.GroupId.Hex())

	switch this.EType {
	case VT_NEW_POST, VT_NEW_KNOWLEDGE, VT_NEW_QORTEX_BROADCAST, VT_NEW_QORTEX_FEEDBACK:
		url = fmt.Sprintf("%v/entry/%v", baseUrl, this.RootId.Hex())
	case VT_NEW_COMMENT:
		url = fmt.Sprintf("%v/entry/%v/cid/%v", baseUrl, this.RootId.Hex(), this.EntryId.Hex())
	}

	return
}

func NewNotification(toUser, fromUser *users.EmbedUser, eType string,
	apiEntry *qortexapi.Entry, createdAt time.Time) *Notification {

	notifi := &Notification{
		UserId:    toUser.Id,
		OrgId:     bson.ObjectIdHex(toUser.OriginalOrgId),
		FromUser:  fromUser,
		EntryId:   bson.ObjectIdHex(apiEntry.Id),
		GroupId:   bson.ObjectIdHex(apiEntry.GroupId),
		Title:     apiEntry.Title,
		Content:   utils.CutString(apiEntry.Content, CONTENT_STRING_LENGTH),
		EType:     eType,
		CreatedAt: createdAt,
	}

	if apiEntry.RootId != "" {
		notifi.RootId = bson.ObjectIdHex(apiEntry.RootId)
	} else {
		notifi.RootId = notifi.EntryId
	}

	return notifi
}

func PutNotificationsIntoChan(db *mgodb.Database, orgId, userId bson.ObjectId, before time.Time,
	limit int, notifisChan chan<- []*Notification) {

	var notifis []*Notification

	defer func() {
		if x := recover(); x != nil {
			utils.PrintfStackAndError("Get Notification Items error: %s", x)
			notifisChan <- notifis
		}
	}()

	notifis, _ = GetSomeNotifications(db, orgId, userId, before, limit)

	notifisChan <- notifis

	return
}

func GetNotificationsFromChan(notifisChan <-chan []*Notification, length int) (notifis []*Notification) {
	for i := 0; i < length; i++ {
		items := <-notifisChan
		notifis = append(notifis, items...)
	}

	return
}
