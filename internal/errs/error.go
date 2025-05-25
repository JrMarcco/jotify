package errs

import "errors"

var (
	ErrInvalidParam               = errors.New("[jotify] invalid param")
	ErrBizIdNotFound              = errors.New("[jotify] biz id not found")
	ErrBizConfNotFound            = errors.New("[jotify] biz config not found")
	ErrFailedSendNotification     = errors.New("[jotify] failed to send notification")
	ErrInvalidChannel             = errors.New("[jotify] invalid channel")
	ErrInvalidSendStrategy        = errors.New("[jotify] invalid send strategy")
	ErrNoAvailableFailoverService = errors.New("[jotify] no service needs to be take over")
	ErrInsufficientQuota          = errors.New("[jotify] insufficient quota")
	ErrFailedToCreateCallbackLog  = errors.New("[jotify] failed to create callback log")
)
