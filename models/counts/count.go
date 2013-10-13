package counts

import (
	"github.com/kobeld/qortex-realtime/global"
	"github.com/kobeld/qortex-realtime/models/notifications"
	"github.com/sunfmin/mgodb"
	"github.com/theplant/qortex/entries"
	"github.com/theplant/qortex/groups"
	"github.com/theplant/qortex/utils"
	"github.com/theplant/qortexapi"
	"labix.org/v2/mgo/bson"
	"sync"
)

var mu sync.Mutex

type MyCount struct {
	Id             bson.ObjectId `bson:"_id"`
	UserId         bson.ObjectId
	OrganizationId bson.ObjectId `bson:",omitempty"` // Qortex Support needs this

	FollowedTotalCount      int
	OfflineMessageCount     int
	ActiveTasksCount        int
	NotificationUnreadCount int

	GroupCounts []*GroupCount
}

type GroupCount struct {
	GroupId     bson.ObjectId
	UnreadCount int
	Following   bool
}

func (this *MyCount) MakeId() interface{} {
	if this.Id == "" {
		this.Id = bson.NewObjectId()
	}
	return this.Id
}

func ResetCount(db *mgodb.Database, orgId, userId, groupId bson.ObjectId) (myCount *MyCount, err error) {
	mu.Lock()
	defer mu.Unlock()

	myCount = findOrNewMyCount(db, orgId, userId)

	if err = myCount.sumAndSave(db, orgId, userId, groupId); err != nil {
		utils.PrintStackAndError(err)
		return
	}

	if !global.IsQortexSupportDB(db) {
		return
	}

	// If it is Qortex Support group, tt may be more than one MyCount in Qortex Global DB,
	// which is maintaining user Qortex Support count in different organizations.
	utils.Go(func() {
		qortexSupportCounts, err := FindMyCountsByUseId(db, userId)
		if err != nil {
			// Don't return
			utils.PrintStackAndError(err)
		}

		for _, count := range qortexSupportCounts {
			count.sumAndSave(db, orgId, userId, count.OrganizationId)
		}
	})

	return
}

func SumAndGetAllDbCount(userDbs []*mgodb.Database, orgId, userId bson.ObjectId) *MyCount {
	countChan := make(chan *MyCount)
	for _, db := range userDbs {
		go putCountIntoChan(db, orgId, userId, countChan)
	}
	allCounts := getCountsFromChan(countChan, len(userDbs))

	count := new(MyCount)
	for _, item := range allCounts {
		count.UserId = item.UserId
		count.FollowedTotalCount = count.FollowedTotalCount + item.FollowedTotalCount
		count.OfflineMessageCount = count.OfflineMessageCount + item.OfflineMessageCount
		count.ActiveTasksCount = count.ActiveTasksCount + item.ActiveTasksCount
		count.NotificationUnreadCount = count.NotificationUnreadCount + item.NotificationUnreadCount
		count.GroupCounts = append(count.GroupCounts, item.GroupCounts...)
	}

	return count
}

func (this *MyCount) ToApiCount() *qortexapi.MyCount {
	apiGroupCounts := []*qortexapi.GroupCount{}

	for _, gc := range this.GroupCounts {
		apiGroupCounts = append(apiGroupCounts, &qortexapi.GroupCount{
			GroupId:     gc.GroupId.Hex(),
			UnreadCount: gc.UnreadCount,
		})
	}

	return &qortexapi.MyCount{
		UserId:                  this.UserId.Hex(),
		FollowedUnreadCount:     this.FollowedTotalCount,
		OfflineMessageCount:     this.OfflineMessageCount,
		ActiveTasksCount:        this.ActiveTasksCount,
		NotificationUnreadCount: this.NotificationUnreadCount,
		GroupCounts:             apiGroupCounts,
	}
}

// ----- Private Methods -----

func (this *MyCount) getOrNewGroupCount(db *mgodb.Database, groupId bson.ObjectId) (groupCount *GroupCount) {

	for _, gc := range this.GroupCounts {
		if gc.GroupId == groupId {
			return gc
		}
	}

	groupCount = &GroupCount{
		GroupId:   groupId,
		Following: groups.UserFollowingGroup(db, this.UserId, groupId),
	}
	this.GroupCounts = append(this.GroupCounts, groupCount)
	return
}

func (this *MyCount) sumAndSave(db *mgodb.Database, orgId, userId, groupId bson.ObjectId) (err error) {
	this.sumGroupCount(db, orgId, userId, groupId)
	this.sumMyFeedNum(db, userId)
	this.sumNotificationNum(db, orgId, userId)

	if err = this.Save(db); err != nil {
		utils.PrintStackAndError(err)
		return
	}
	return
}

func (this *MyCount) sumGroupCount(db *mgodb.Database, orgId, userId, groupId bson.ObjectId) {

	groupCount := this.getOrNewGroupCount(db, groupId)

	query := bson.M{
		"readerid":  userId,
		"groupid":   groupId,
		"hasunread": true,
	}
	if global.IsQortexSupportDB(db) {
		query["organizationid"] = orgId
	}

	//Find all unread Entries Readers
	entriesReaders, err := entries.FindAllEntriesReaders(db, query)
	if err != nil {
		utils.PrintStackAndError(err)
		return
	}

	unreadCount := 0
	for _, er := range entriesReaders {
		unreadCount = unreadCount + len(er.UnreadIds)
	}

	groupCount.UnreadCount = unreadCount
}

func (this *MyCount) sumMyFeedNum(db *mgodb.Database, userId bson.ObjectId) {
	unfollowedGroupIds := []bson.ObjectId{}
	followedUnread := 0

	for _, groupCount := range this.GroupCounts {
		if groupCount.Following {
			followedUnread = followedUnread + groupCount.UnreadCount
		} else {
			unfollowedGroupIds = append(unfollowedGroupIds, groupCount.GroupId)
		}
	}

	// Count those being notified entries in unfollowed groups
	if len(unfollowedGroupIds) > 0 {
		entriesReaders, _ := entries.FindNotifiedEntriesReaderByUserAndGroupIds(db, userId, unfollowedGroupIds)
		for _, er := range entriesReaders {
			followedUnread = followedUnread + len(er.UnreadIds)
		}
	}

	this.FollowedTotalCount = followedUnread
}

func (this *MyCount) sumNotificationNum(db *mgodb.Database, orgId, userId bson.ObjectId) {

	query := bson.M{
		"userid": userId,
		"readat": bson.M{"$exists": false},
	}

	if global.IsQortexSupportDB(db) {
		query["organizationid"] = orgId
	}

	unreadNum, err := notifications.CountNotifications(db, query)
	if err != nil {
		utils.PrintStackAndError(err)
		return
	}

	this.NotificationUnreadCount = unreadNum

	// If it is qortex support db, no need to count the task number
	if global.IsQortexSupportDB(db) {
		return
	}

	query = bson.M{
		"touserids":             userId,
		"task.completedat":      bson.M{"$exists": false},
		"task.byuserid":         bson.M{"$exists": true},
		"task.completeduserids": bson.M{"$ne": userId}}

	tasksNum, err := entries.CountEntry(db, query)
	if err != nil {
		utils.PrintStackAndError(err)
		return
	}
	this.ActiveTasksCount = tasksNum
}

func findOrNewMyCount(db *mgodb.Database, orgId, userId bson.ObjectId) (myCount *MyCount) {
	myCount = FindMyCountByOrgAndUserId(db, orgId, userId)
	if myCount != nil {
		return
	}
	myCount = &MyCount{UserId: userId, OrganizationId: orgId}
	return
}

func putCountIntoChan(db *mgodb.Database, orgId, userId bson.ObjectId, countChan chan<- *MyCount) {
	var count *MyCount

	defer func() {
		if x := recover(); x != nil {
			utils.PrintfStackAndError("Get MyCount error: %s", x)
			countChan <- count
		}
	}()

	countChan <- FindMyCountByOrgAndUserId(db, orgId, userId)
}

func getCountsFromChan(countChan <-chan *MyCount, length int) (myCounts []*MyCount) {
	for i := 0; i < length; i++ {
		count := <-countChan
		if count != nil {
			myCounts = append(myCounts, count)
		}
	}

	return
}
