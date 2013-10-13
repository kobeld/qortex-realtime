package counts

import (
	"github.com/kobeld/qortex-realtime/global"
	"github.com/sunfmin/mgodb"
	"labix.org/v2/mgo/bson"
)

const (
	MY_COUNTS = "my_counts"
)

func (this *MyCount) Save(db *mgodb.Database) error {
	return db.Save(MY_COUNTS, this)
}

func FindMyCountByOrgAndUserId(db *mgodb.Database, orgId, userId bson.ObjectId) (myCount *MyCount) {
	query := bson.M{"userid": userId}

	// Qortex Support Database
	if global.IsQortexSupportDB(db) {
		query["organizationid"] = orgId
	}

	myCount, _ = FindOneMyCount(db, query)
	return
}

func FindMyCountsByUseId(db *mgodb.Database, userId bson.ObjectId) (myCounts []*MyCount, err error) {
	return FindAllMyCounts(db, bson.M{"userid": userId})

}

func FindOneMyCount(db *mgodb.Database, query bson.M) (myCount *MyCount, err error) {
	db.FindOne(MY_COUNTS, query, &myCount)
	return
}

func FindAllMyCounts(db *mgodb.Database, query bson.M) (myCounts []*MyCount, err error) {
	db.FindAll(MY_COUNTS, query, &myCounts)
	return
}
