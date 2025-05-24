package grpc

import notificationv1 "github.com/JrMarcco/jotify-api/api/notification/v1"

type NotificationServer struct {
	notificationv1.UnimplementedNotificationServiceServer
	notificationv1.UnimplementedNotificationQueryServiceServer
}

func NewNotificationServer() *NotificationServer {
	return &NotificationServer{}
}
