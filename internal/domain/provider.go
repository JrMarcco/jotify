package domain

import (
	"fmt"

	"github.com/JrMarcco/jotify/internal/errs"
)

// Channel 发送渠道（邮件/短信/站内信）
type Channel string

const (
	ChannelEmail Channel = "email"
	ChannelSMS   Channel = "sms"
	ChannelApp   Channel = "app"
)

func (c Channel) String() string {
	return string(c)
}

func (c Channel) Validate() bool {
	return c == ChannelEmail || c == ChannelSMS || c == ChannelApp
}

func (c Channel) IsSMS() bool {
	return c == ChannelSMS
}

func (c Channel) IsEmail() bool {
	return c == ChannelEmail
}

func (c Channel) IsApp() bool {
	return c == ChannelApp
}

// ProviderStatus 供应商状态
type ProviderStatus string

const (
	ProviderStatusActive   ProviderStatus = "active"
	ProviderStatusInactive ProviderStatus = "inactive"
)

func (s ProviderStatus) String() string {
	return string(s)
}

// Provider 供应商
type Provider struct {
	Id      uint64
	Name    string
	Channel Channel

	Endpoint string
	RegionId string

	AppId     string
	ApiKey    string
	ApiSecret string

	Weight     int32
	QpsLimit   int32
	DailyLimit int32

	CallbackUrl string
	Status      ProviderStatus
}

func (p *Provider) Validate() error {
	if !p.Channel.Validate() {
		return fmt.Errorf("%w invalid channel", errs.ErrInvalidParam)
	}

	if p.Name == "" {
		return fmt.Errorf("%w provider name should not be empty", errs.ErrInvalidParam)
	}

	if p.Endpoint == "" {
		return fmt.Errorf("%w api endpoint should not be empty", errs.ErrInvalidParam)
	}

	if p.ApiKey == "" {
		return fmt.Errorf("%w api key should not be empty", errs.ErrInvalidParam)
	}

	if p.ApiSecret == "" {
		return fmt.Errorf("%w api secret should not be empty", errs.ErrInvalidParam)
	}

	if p.Weight <= 0 {
		return fmt.Errorf("%w weight should not be greater than 0", errs.ErrInvalidParam)
	}

	if p.QpsLimit <= 0 {
		return fmt.Errorf("%w qps limit should not be greater than 0", errs.ErrInvalidParam)
	}

	if p.DailyLimit <= 0 {
		return fmt.Errorf("%w daily limit should not be greater than 0", errs.ErrInvalidParam)
	}

	return nil
}
