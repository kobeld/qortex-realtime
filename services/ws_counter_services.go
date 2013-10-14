package services

import (
	"github.com/kobeld/qortex-realtime/models/counts"
	"github.com/theplant/qortex/utils"
	"github.com/theplant/qortexapi"
)

const (
	COUNTER_READ_ENTRY        = "Counter.ReadEntry"
	COUNTER_READ_MESSAGE      = "Counter.ReadMyMessage"
	COUNTER_READ_NOTIFICATION = "Counter.ReadNotificationItem"
	COUNTER_REFRESH           = "Counter.Refresh"
)

// Counter related reply data that
// Refresh, ReadEntry and ReadNotificaiton all using
type CountNotification struct {
	Method           string
	GroupId          string
	NewEntry         bool
	EntryId          string
	DelType          string
	MyCount          *qortexapi.MyCount
	NewMessageNumber int
}

type Counter int

type RefreshInput struct {
	LoggedInUserId string
	OrganizationId string
}

func (this *Counter) Refresh(input *RefreshInput, reply *CountNotification) (err error) {

	defer func() {
		if x := recover(); x != nil {
			utils.PrintfStackAndError("Error: %+v \n For:", x.(error), input)
		}
	}()

	reply.Method = COUNTER_REFRESH

	activeOrg, onlineUser, err := getActiveOrgAndOnlineUser(input.OrganizationId, input.LoggedInUserId)
	if err != nil {
		utils.PrintStackAndError(err)
		return
	}

	totolCount := counts.SumAndGetAllDbCount(onlineUser.AllDBs(), activeOrg.Organization.Id, onlineUser.User.Id)
	reply.MyCount = totolCount.ToApiCount()
	return
}

// Read Entry struct and methods
type ReaderInput struct {
	EntryId        string
	ReaderId       string
	GroupId        string
	OrganizationId string
	ConversationId string
	NotificationId string
}

func (this *ReaderInput) isValid() bool {
	if this.EntryId == "" || this.ReaderId == "" || this.OrganizationId == "" {
		return false
	}
	return true
}

func (this *ReaderInput) isValidForNotificaiton() bool {
	if this.NotificationId == "" || this.ReaderId == "" || this.OrganizationId == "" {
		return false
	}

	return true
}

func (this *ReaderInput) isReadEntry() bool {
	return this.GroupId != "" && this.ConversationId == ""
}

func (this *ReaderInput) isReadMyMessage() bool {
	return this.GroupId == "" && this.ConversationId != ""
}

func (this *Counter) ReadEntry(input *ReaderInput, reply *CountNotification) (err error) {

	defer func() {
		if x := recover(); x != nil {
			utils.PrintfStackAndError("Read entry error: %+v \n For:", x.(error), input)
		}
	}()

	// Simple validation for those needed fields
	if !input.isValid() {
		return
	}

	_, onlineUser, err := getActiveOrgAndOnlineUser(input.OrganizationId, input.ReaderId)
	if err != nil {
		utils.PrintStackAndError(err)
		return
	}

	var method string
	var myCount *qortexapi.MyCount
	switch {
	case input.isReadEntry():
		method = COUNTER_READ_ENTRY
		// myCount, err = serv.ReadEntry(input.EntryId, input.GroupId)
		myCount, err = ReadEntry(input.OrganizationId, input.ReaderId, input.GroupId, input.EntryId)
	case input.isReadMyMessage():
		method = COUNTER_READ_MESSAGE
		// myCount, err = serv.ReadMyMessage(input.ConversationId)
	default:
		return
	}
	if err != nil {
		utils.PrintStackAndError(err)
		return
	}

	if onlineUser == nil {
		return
	}

	newReply := CountNotification{
		Method:           method,
		EntryId:          input.EntryId,
		GroupId:          input.GroupId,
		MyCount:          myCount,
		NewMessageNumber: onlineUser.ClearNewMessageId(),
	}
	onlineUser.SendReply(newReply)

	return
}

//  Read notification at real-time
func (this *Counter) ReadNotification(input *ReaderInput, reply *CountNotification) (err error) {

	defer func() {
		if x := recover(); x != nil {
			utils.PrintfStackAndError("Read notification error: %+v \n For:", x.(error), input)
		}
	}()

	// Simple validation for those needed fields
	if !input.isValidForNotificaiton() {
		return
	}

	_, onlineUser, err := getActiveOrgAndOnlineUser(input.OrganizationId, input.ReaderId)
	if err != nil {
		utils.PrintStackAndError(err)
		return
	}

	myCount, err := ReadNotification(input.OrganizationId, input.ReaderId, input.GroupId, input.NotificationId)
	if err != nil {
		utils.PrintStackAndError(err)
		return
	}

	if onlineUser == nil {
		return
	}

	newReply := CountNotification{
		Method:  COUNTER_READ_NOTIFICATION,
		MyCount: myCount,
	}
	onlineUser.SendReply(newReply)

	return
}
