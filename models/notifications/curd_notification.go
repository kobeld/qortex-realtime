package notifications

import (
	"github.com/kobeld/qortex-realtime/global"
	"github.com/sunfmin/mgodb"
	"github.com/theplant/qortex/utils"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"time"
)

const (
	NOTIFICATIONS = "notifications"
)

func ReadNotificationById(db *mgodb.Database, itemId bson.ObjectId) (info *mgo.ChangeInfo, err error) {
	selector := bson.M{"_id": itemId}
	changer := bson.M{"$set": bson.M{"readat": time.Now()}}
	return UpdateAllNotifications(db, selector, changer)
}

func ReadNotificationByUserAndEntryId(db *mgodb.Database, readerId, entryId bson.ObjectId, readAt time.Time) (err error) {
	selector := bson.M{"userid": readerId, "entryid": entryId}
	changer := bson.M{"$set": bson.M{"readat": readAt}}
	_, err = UpdateAllNotifications(db, selector, changer)
	return
}

func DeleteNotificationById(db *mgodb.Database, itemId bson.ObjectId) (err error) {
	query := bson.M{"_id": itemId}
	_, err = DeleteNotification(db, query)
	return
}

func GetSomeNotifications(db *mgodb.Database, orgId, userId bson.ObjectId, before time.Time,
	limit int) (notifs []*Notification, err error) {

	query := bson.M{
		"userid":    userId,
		"readat":    bson.M{"$exists": false},
		"createdat": bson.M{"$lt": before},
	}

	// For getting Qortex Support notification items
	if global.IsQortexSupportDB(db) {
		query["orgid"] = orgId
	}

	db.CollectionDo(NOTIFICATIONS, func(rc *mgo.Collection) {
		err = rc.Find(query).Sort("-created").Limit(limit).All(&notifs)
	})

	if err != nil {
		utils.PrintStackAndError(err)
		return
	}

	if len(notifs) == limit {
		return
	}

	var readedItems []*Notification
	query["readat"] = bson.M{"$exists": true}
	db.CollectionDo(NOTIFICATIONS, func(rc *mgo.Collection) {
		rc.Find(query).Sort("-created").Limit(limit - len(notifs)).All(&readedItems)
	})

	if err != nil {
		utils.PrintStackAndError(err)
		return
	}

	if len(readedItems) > 0 {
		notifs = append(notifs, readedItems...)
	}

	return
}

func GetUnreadNotifications(db *mgodb.Database, userId, orgId bson.ObjectId, after time.Time,
	limit int) (notifs []*Notification, err error) {

	query := bson.M{
		"userid":    userId,
		"readat":    bson.M{"$exists": false},
		"createdat": bson.M{"$gt": after},
	}

	if global.IsQortexSupportDB(db) {
		query["orgid"] = orgId
	}

	db.CollectionDo(NOTIFICATIONS, func(rc *mgo.Collection) {
		q := rc.Find(query).Sort("-createdat")
		if limit > 0 {
			q = q.Limit(limit)
		}

		err = q.All(&notifs)
	})

	if err != nil {
		utils.PrintStackAndError(err)
		return
	}

	return

	// What this used for?
	// for i, _ := range notifs {
	// notifs[i].Database = db
	// }
}

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

func DeleteNotification(db *mgodb.Database, query bson.M) (info *mgo.ChangeInfo, err error) {
	db.CollectionDo(NOTIFICATIONS, func(rc *mgo.Collection) {
		info, err = rc.RemoveAll(query)
	})
	return
}
