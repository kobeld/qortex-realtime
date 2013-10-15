package notifications

import (
	"github.com/theplant/qortex/users"
	"github.com/theplant/qortexapi"
)

// Note:
// 	Notification type is a subset of Event type,
// 	and here is the only place to declare new type
const (
	VT_DEFAULT string = "event_000"
	VT_REMIND         = "event_001"

	VT_LIKE        = "event_010"
	VT_REMOVE_LIKE = "event_011"

	VT_NEW_POST              = "event_020"
	VT_POST_NEED_ACK         = "event_021"
	VT_POST_ONE_ACKED        = "event_022"
	VT_POST_OWNER_CANCEL_ACK = "event_023"

	VT_NEW_KNOWLEDGE = "event_030"

	VT_NEW_COMMENT              = "event_040"
	VT_COMMENT_NEED_ACK         = "event_041"
	VT_COMMENT_ONE_ACKED        = "event_042"
	VT_COMMENT_OWNER_CANCEL_ACK = "event_043"

	VT_NEW_TODO         = "event_060"
	VT_ONE_FIN_TODO     = "event_061"
	VT_OWNER_CLOSE_TODO = "event_062"

	VT_NEW_QORTEX_BROADCAST     = "event_070"
	VT_NEW_SHARED_REQUEST       = "event_071"
	VT_ACCEPT_SHARED_REQUEST    = "event_072"
	VT_REJECT_SHARED_REQUEST    = "event_073"
	VT_FORWARDED_SHARED_REQUEST = "event_074"
	VT_NEW_QORTEX_FEEDBACK      = "event_075"
	VT_REJECT_BEFORE_FORWARDING = "event_076"
	VT_CANCEL_BEFORE_FORWARDING = "event_077"
	VT_CANCEL_SHARED_REQUEST    = "event_078"
	VT_STOP_SHARING_GROUP       = "event_079"

	VT_NEW_INNER_MESSAGE = "event_080"

	VT_DELETE_ENTRY_ALL      = "event_101"
	VT_UPDATE_ENTRY_VERSION  = "event_102"
	VT_DELETE_QORTEX_SUPPORT = "event_103"
)

type Event struct {
	ToUser       *users.EmbedUser
	VType        string
	ShowNewBar   bool
	Notification *Notification
}

func NewEvent(toUser *users.EmbedUser, vType string, showNewBar bool) *Event {

	return &Event{
		ToUser:     toUser,
		VType:      vType,
		ShowNewBar: showNewBar,
	}
}

// Decide the event type above by the Api Entry
func DecideEventType(apiEntry *qortexapi.Entry) string {
	switch {
	case apiEntry.IsKnowledgeBase:
		return VT_NEW_KNOWLEDGE
	case apiEntry.IsComment:
		return VT_NEW_COMMENT
	case apiEntry.IsPost:
		return VT_NEW_POST
	case apiEntry.IsInnerMessage:
		return VT_NEW_INNER_MESSAGE
	default:
		return VT_DEFAULT
	}
}
