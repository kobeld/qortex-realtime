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

type Notification struct {
	Id        bson.ObjectId `bson:"_id"`
	UserId    bson.ObjectId
	OrgId     bson.ObjectId // Original Org Id
	GroupId   bson.ObjectId
	EntryId   bson.ObjectId // The entry caused this notification
	FromUser  *users.EmbedUser
	Title     string
	Content   string
	RootId    bson.ObjectId `bson:",omitempty"` // Comment on Entry Id
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
	case VT_NEW_POST:
		url = fmt.Sprintf("%v/entry/%v", baseUrl, this.RootId.Hex())
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
		Content:   apiEntry.Content,
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
