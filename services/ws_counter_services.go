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
