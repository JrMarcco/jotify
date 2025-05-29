package errs

import (
	"errors"
)

var (
	ErrInvalidParam              = errors.New("[jotify] invalid param")
	ErrBizIdNotFound             = errors.New("[jotify] biz id not found")
	ErrBizConfNotFound           = errors.New("[jotify] biz config not found")
	ErrChannelTplNotFound        = errors.New("[jotify] channel template not found")
	ErrChannelTplVersionNotFound = errors.New("[jotify] channel template version not found")
	ErrNotApprovedTplVersion     = errors.New("[jotify] channel template version is not approved")
	ErrFailedSendNotification    = errors.New("[jotify] failed to send notification")
	ErrInvalidChannel            = errors.New("[jotify] invalid channel")
	ErrInvalidSendStrategy       = errors.New("[jotify] invalid send strategy")
	ErrInsufficientQuota         = errors.New("[jotify] insufficient quota")
	ErrFailedToCreateCallbackLog = errors.New("[jotify] failed to create callback log")
	ErrNoAvailableProvider       = errors.New("[jotify] no available provider")
	ErrFailedToSendNotification  = errors.New("[jotify] failed to send notification")
)
