package errs

import "errors"

var (
	ErrInvalidParam               = errors.New("[jotify] invalid param")
	ErrBizIdNotFound              = errors.New("[jotify] biz id not found")
	ErrFailedSendNotification     = errors.New("[jotify] failed to send notification")
	ErrInvalidChannel             = errors.New("[jotify] invalid channel")
	ErrInvalidSendStrategy        = errors.New("[jotify] invalid send strategy")
	ErrNoAvailableFailoverService = errors.New("[jotify] no service needs to be take over")
)
