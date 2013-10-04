package services

import (
	"github.com/theplant/qortex/utils"
	"github.com/theplant/qortexapi"
)

const (
	COUNTER_READ_ENTRY        = "Counter.ReadEntry"
	COUNTER_READ_MESSAGE      = "Counter.ReadMyMessage"
	COUNTER_READ_NOTIFICATION = "Counter.ReadNotificationItem"
	COUNTER_REFRESH           = "Counter.Refresh"
)

type Counter int

type RefreshInput struct {
	LoggedInUserId string
	OrganizationId string
}

type CountNotification struct {
	Method           string
	GroupId          string
	NewEntry         bool
	EntryId          string
	DelType          string
	MyCount          *qortexapi.MyCount
	NewMessageNumber int
}

func (c *Counter) Refresh(input *RefreshInput, reply *CountNotification) (err error) {

	defer func() {
		if x := recover(); x != nil {
			utils.PrintfStackAndError("Error: %+v \n For:", x.(error), input)
		}
	}()

	reply.Method = COUNTER_REFRESH
	serv, err := MakeWsService(input.OrganizationId, input.LoggedInUserId)
	if err != nil {
		utils.PrintStackAndError(err)
		return
	}

	reply.MyCount, err = serv.GetMyCount()
	if err != nil {
		utils.PrintStackAndError(err)
		return
	}
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

func (c *Counter) ReadEntry(input *ReadEntryInput, reply *CountNotification) (err error) {

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
