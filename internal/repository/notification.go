package repository

import (
	"github.com/JrMarcco/jotify/internal/repository/dao"
	"go.uber.org/zap"
)

type NotificationRepo interface {
}

var _ NotificationRepo = (*DefaultNotifRepo)(nil)

type DefaultNotifRepo struct {
	notifDAO dao.NotificationDAO
	logger   *zap.Logger
}

func NewDefaultNotifRepo(notifDAO dao.NotificationDAO, logger *zap.Logger) *DefaultNotifRepo {
	return &DefaultNotifRepo{
		notifDAO: notifDAO,
		logger:   logger,
	}
}
