package notifications

import (
	"github.com/sunfmin/mgodb"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
)

const (
	NOTIFICATIONS = "notifications"
)

func SaveNotifications(db *mgodb.Database, notifis []*Notification) (err error) {
	if len(notifis) == 0 {
		return
	}

	nis := []interface{}{}
	for _, ni := range notifis {
		ni.MakeId()
		nis = append(nis, ni)
	}

	db.CollectionDo(NOTIFICATIONS, func(rc *mgo.Collection) {
		err = rc.Insert(nis...)
	})
	return
}

func CountNotifications(db *mgodb.Database, query bson.M) (num int, err error) {
	db.CollectionDo(NOTIFICATIONS, func(c *mgo.Collection) {
		num, err = c.Find(query).Count()
	})
	return
}

func UpdateAllNotifications(db *mgodb.Database, selector, changer bson.M) (info *mgo.ChangeInfo, err error) {
	db.CollectionDo(NOTIFICATIONS, func(rc *mgo.Collection) {
		info, err = rc.UpdateAll(selector, changer)
	})

	return
}
