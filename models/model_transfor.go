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
		Id:       notifi.Id.Hex(),
		RootId:   notifi.RootId.Hex(),
		GroupId:  notifi.GroupId.Hex(),
		Type:     notifi.EType,
		HasRead:  !notifi.ReadAt.IsZero(),
		Title:    global.StringToHtml(notifi.Title),
		Content:  global.StringToHtml(notifi.Content),
		Link:     template.HTMLAttr(notifi.Link()),
		FromUser: ToApiEmbedUser(notifi.FromUser),
		IsRoot:   notifi.EntryId == notifi.RootId,
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
			Avatar: embedUser.ExternalAvatar(),
			// Not including other fields that not using in the popup
		}
	}

	return apiEmbedUser
}
