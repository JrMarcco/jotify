package errs

import (
	"errors"
)

var (
	ErrInvalidParam        = errors.New("[jotify] invalid param")
	ErrInvalidChannel      = errors.New("[jotify] invalid channel")
	ErrInvalidSendStrategy = errors.New("[jotify] invalid send strategy")

	ErrBizIdNotFound             = errors.New("[jotify] biz id not found")
	ErrBizConfNotFound           = errors.New("[jotify] biz config not found")
	ErrChannelTplNotFound        = errors.New("[jotify] channel template not found")
	ErrChannelTplVersionNotFound = errors.New("[jotify] channel template version not found")
	ErrNotificationNotFound      = errors.New("[jotify] notification not found")
	ErrFailedSendNotification    = errors.New("[jotify] failed to send notification")

	ErrNotApprovedTplVersion = errors.New("[jotify] channel template version is not approved")
	ErrNotAvailableProvider  = errors.New("[jotify] not available provider")

	ErrInsufficientQuota = errors.New("[jotify] insufficient quota")

	ErrFailedToCreateCallbackLog = errors.New("[jotify] failed to create callback log")
	ErrFailedToSendNotification  = errors.New("[jotify] failed to send notification")

	ErrDuplicateNotificationId = errors.New("[jotify] duplicate notification id")

	ErrAcquireExceedLimit = errors.New("[jotify] acquire resource exceed the limit")

	ErrEventThresholdExceeded = errors.New("[jotify] error event threshold exceeded")
)
