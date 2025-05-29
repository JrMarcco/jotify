package client

import (
	"fmt"
	"strings"

	"github.com/JrMarcco/jotify/internal/errs"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	sms "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/sms/v20210111"
)

var _ SmsClient = (*TencentSmsClient)(nil)

// TencentSmsClient 腾讯云短信客户端
//
// SDK 安装：
//
//	go get -v -u github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/sms/v20210111
type TencentSmsClient struct {
	client *sms.Client
	appId  *string
}

func (tc *TencentSmsClient) Send(req SendReq) (SendResp, error) {
	if len(req.PhoneNumbers) == 0 {
		return SendResp{}, fmt.Errorf("%w: phone number should not be empty", errs.ErrInvalidParam)
	}

	phoneNumberSet := make([]*string, len(req.PhoneNumbers))
	for _, phoneNumber := range req.PhoneNumbers {
		fullPhoneNum := phoneNumber
		if !strings.HasPrefix(phoneNumber, "+") {
			fullPhoneNum = "+86" + phoneNumber
		}
		phoneNumPtr := fullPhoneNum
		phoneNumberSet = append(phoneNumberSet, &phoneNumPtr)
	}

	request := sms.NewSendSmsRequest()
	request.PhoneNumberSet = phoneNumberSet
	request.SmsSdkAppId = tc.appId
	request.TemplateId = &req.TemplateId
	request.SignName = &req.SignName

	if req.TemplateParams != nil {
		templateParamSet := make([]*string, len(req.TemplateParams))
		for _, param := range req.TemplateParams {
			tplParam := param
			templateParamSet = append(templateParamSet, &tplParam)
		}
		request.TemplateParamSet = templateParamSet
	}

	res, err := tc.client.SendSms(request)
	if err != nil {
		return SendResp{}, fmt.Errorf("%w: %w", ErrFailedToSendSms, err)
	}

	if len(res.Response.SendStatusSet) == 0 {
		return SendResp{}, fmt.Errorf("%w: no response from tencent", ErrFailedToSendSms)
	}

	sendResp := SendResp{
		RequestId:    *res.Response.RequestId,
		PhoneNumbers: make(map[string]SendRespStatus),
	}

	for _, status := range res.Response.SendStatusSet {
		phoneNumber := strings.TrimPrefix(*status.PhoneNumber, "+86")
		sendResp.PhoneNumbers[phoneNumber] = SendRespStatus{
			Code:    *status.Code,
			Message: *status.Message,
		}
	}
	return sendResp, nil
}

func (tc *TencentSmsClient) CreateTemplate(req CreateTplReq) (CreateTplResp, error) {
	//TODO implement me
	panic("implement me")
}

func NewTencentSmsClient(regionId, secretId, secretKey, appId string) *TencentSmsClient {
	client, err := sms.NewClient(common.NewCredential(secretId, secretKey), regionId, profile.NewClientProfile())
	if err != nil {
		panic(err)
	}
	return &TencentSmsClient{
		client: client,
		appId:  &appId,
	}
}
