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

	userId, err := utils.ToObjectId(input.LoggedInUserId)
	if err != nil {
		utils.PrintStackAndError(err)
		return
	}

	activeOrg, err := MyActiveOrg(input.OrganizationId)
	if err != nil {
		utils.PrintStackAndError(err)
		return
	}

	onlineUser, err := activeOrg.GetOnlineUserById(userId)
	if err != nil {
		utils.PrintStackAndError(err)
		return
	}

	totolCount := counts.SumAndGetAllDbCount(onlineUser.AllDBs(), activeOrg.Organization.Id, userId)

	reply.MyCount = totolCount.ToApiCount()
	return
}

// Read Entry struct and methods
type ReadEntryInput struct {
	EntryId        string
	ReaderId       string
	GroupId        string
	OrganizationId string
	ConversationId string
}

func (this *ReadEntryInput) isValid() bool {
	if this.EntryId == "" || this.ReaderId == "" || this.OrganizationId == "" {
		return false
	}
	return true
}

func (this *ReadEntryInput) isReadEntry() bool {
	return this.GroupId != "" && this.ConversationId == ""
}

func (this *ReadEntryInput) isReadMyMessage() bool {
	return this.GroupId == "" && this.ConversationId != ""
}

func (this *Counter) ReadEntry(input *ReadEntryInput, reply *CountNotification) (err error) {

	defer func() {
		if x := recover(); x != nil {
			utils.PrintfStackAndError("Error: %+v \n For:", x.(error), input)
		}
	}()

	if !input.isValid() {
		return
	}

	var myCount *qortexapi.MyCount
	serv, err := MakeWsService(input.OrganizationId, input.ReaderId)
	if err != nil {
		utils.PrintStackAndError(err)
		return
	}

	var method string
	switch {
	case input.isReadEntry():
		method = COUNTER_READ_ENTRY
		myCount, err = serv.ReadEntry(input.EntryId, input.GroupId)
	case input.isReadMyMessage():
		method = COUNTER_READ_MESSAGE
		myCount, err = serv.ReadMyMessage(input.ConversationId)
	default:
		return
	}
	if err != nil {
		utils.PrintStackAndError(err)
		return
	}

	if serv.OnlineUser == nil {
		return
	}

	newReply := CountNotification{
		Method:           method,
		EntryId:          input.EntryId,
		GroupId:          input.GroupId,
		MyCount:          myCount,
		NewMessageNumber: serv.OnlineUser.ClearNewMessageId(),
	}
	serv.OnlineUser.SendReply(newReply)

	return
}

// Read the red notificaiton and realtime push the result to client
type ReadNotificationInput struct {
	NotificationItemId string
	ReaderId           string
	GroupId            string
	OrganizationId     string
}

func (this *Counter) ReadNotificationItem(input *ReadNotificationInput, reply *CountNotification) (err error) {

	defer func() {
		if x := recover(); x != nil {
			utils.PrintfStackAndError("Error: %+v \n For:", x.(error), input)
		}
	}()

	var myCount *qortexapi.MyCount
	serv, err := MakeWsService(input.OrganizationId, input.ReaderId)
	if err != nil {
		utils.PrintStackAndError(err)
		return
	}
	if myCount, err = serv.ReadNotificationItem(input.NotificationItemId, input.GroupId); err != nil {
		utils.PrintStackAndError(err)
		return
	}

	if serv.OnlineUser == nil {
		return
	}

	newReply := CountNotification{
		Method:           COUNTER_READ_NOTIFICATION,
		MyCount:          myCount,
		NewMessageNumber: len(serv.OnlineUser.NewMessageIds),
	}
	serv.OnlineUser.SendReply(newReply)

	return
}
