package models

import (
	"github.com/kobeld/qortex-realtime/global"
	"github.com/kobeld/qortex-realtime/models/notifications"
	"github.com/theplant/qortex/users"
	"github.com/theplant/qortexapi"
	"html/template"
)

func ToApiNotificationItem(notifi *notifications.Notification) (apiNotification *qortexapi.NotificationItem) {
	apiNotification = &qortexapi.NotificationItem{
		Id:        notifi.Id.Hex(),
		GroupId:   notifi.GroupId.Hex(),
		Type:      notifi.EType,
		HasRead:   !notifi.ReadAt.IsZero(),
		HtmlTitle: global.StringToHtml(notifi.Title),
		Link:      template.HTMLAttr(notifi.Link()),
		FromUser:  ToApiEmbedUser(notifi.FromUser),
	}

	return
}

func ToApiNotificationItems(notifis []*notifications.Notification) (apiNotifis []*qortexapi.NotificationItem) {
	for _, notifi := range notifis {
		apiNotifis = append(apiNotifis, ToApiNotificationItem(notifi))
	}
	return
}

func ToApiEmbedUser(embedUser *users.EmbedUser) *qortexapi.EmbedUser {
	apiEmbedUser := new(qortexapi.EmbedUser)

	if embedUser != nil {
		apiEmbedUser = &qortexapi.EmbedUser{
			Id:     embedUser.Id.Hex(),
			Name:   embedUser.Name,
			Email:  embedUser.Email,
			Avatar: embedUser.Avatar,
			// Not including other fields that not using in the popup
		}
	}

	return apiEmbedUser
}
