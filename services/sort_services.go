package services

import (
	"github.com/kobeld/qortex-realtime/models/notifications"
)

const MOVE_FORWARD = 1000000000000000000

type NotifItems []*notifications.Notification

type ByNotifiedAt struct {
	NotifItems
}

func (s NotifItems) Len() int { return len(s) }

func (s NotifItems) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

func (s ByNotifiedAt) Less(i, j int) bool {
	return s.NotifItems[i].CreatedAt.UnixNano() > s.NotifItems[j].CreatedAt.UnixNano()
}

type ByNewAndNotifiedAt struct{ NotifItems }

func (s ByNewAndNotifiedAt) Less(i, j int) bool {
	ic := s.NotifItems[i].CreatedAt.UnixNano()
	jc := s.NotifItems[j].CreatedAt.UnixNano()

	if s.NotifItems[i].ReadAt.IsZero() {
		ic = ic + MOVE_FORWARD
	}
	if s.NotifItems[j].ReadAt.IsZero() {
		jc = jc + MOVE_FORWARD
	}
	return ic > jc
}
