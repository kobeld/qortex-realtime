package services

import (
	"github.com/kobeld/qortex-realtime/global"
	"github.com/kobeld/qortex-realtime/models"
	"github.com/kobeld/qortex-realtime/models/notifications"
	"github.com/sunfmin/mgodb"
	"github.com/theplant/qortexapi"
	"labix.org/v2/mgo/bson"
	"sort"
)

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
