package notifications

import (
	"github.com/theplant/qortex/users"
	"github.com/theplant/qortexapi"
	"labix.org/v2/mgo/bson"
	"time"
)

type Notification struct {
	Id        bson.ObjectId `bson:"_id"`
	UserId    bson.ObjectId
	FromUser  *users.EmbedUser
	EntryId   bson.ObjectId // The entry caused this notification
	Title     string
	Content   string
	RootId    bson.ObjectId // Comment on Entry Id
	EType     string
	ReadAt    time.Time
	CreatedAt time.Time
}

func (this *Notification) MakeId() interface{} {
	if this.Id == "" {
		this.Id = bson.NewObjectId()
	}
	return this.Id
}

func NewNotification(toUser, fromUser *users.EmbedUser, eType string,
	apiEntry *qortexapi.Entry, createdAt time.Time) *Notification {

	notifi := &Notification{
		UserId:    toUser.Id,
		FromUser:  fromUser,
		EntryId:   bson.ObjectIdHex(apiEntry.Id),
		Title:     apiEntry.Title,
		Content:   apiEntry.Content,
		EType:     eType,
		CreatedAt: createdAt,
	}

	if apiEntry.RootId != "" {
		notifi.RootId = bson.ObjectIdHex(apiEntry.RootId)
	}

	return notifi
}
