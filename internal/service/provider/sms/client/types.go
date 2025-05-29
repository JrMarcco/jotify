package client

import "fmt"

//go:generate mockgen -source=./types.go -destination=./mock/sms_client.mock.go -package=clientmock -typed SmsClient

var (
	ErrFailedToSendSms = fmt.Errorf("[jotify] failed to send sms")
)

type SmsClient interface {
	Send(req SendReq) (SendResp, error)
	CreateTemplate(req CreateTplReq) (CreateTplResp, error)
}

type SendStatus int32

// SendReq 发送短信请求
type SendReq struct {
	PhoneNumbers   []string
	SignName       string
	TemplateId     string
	TemplateParams map[string]string
}

// SendResp 发送短信响应
type SendResp struct {
	RequestId    string
	PhoneNumbers map[string]SendRespStatus // PhoneNumber -> SendRespStatus
}

// SendRespStatus 短信发送响应状态
//
// 每个手机号对应一个状态
type SendRespStatus struct {
	Code    string
	Message string
}

type TemplateType int32

// CreateTplReq 创建短信模板请求
type CreateTplReq struct {
	TplType    TemplateType
	TplName    string
	TplContent string
	Remark     string
}

// CreateTplResp 创建短信模板响应
type CreateTplResp struct {
	RequestId  string
	TemplateId string
}
