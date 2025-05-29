package provider

import (
	"context"

	"github.com/JrMarcco/jotify/internal/domain"
)

//go:generate mockgen -source=./types.go -destination=./mock/provider.mock.go -package=providermock -typed Provider,Selector,SelectorBuilder

// Provider 供应商接口
type Provider interface {
	Send(ctx context.Context, n domain.Notification) (domain.SendResp, error)
}

// Selector 供应商选择器
type Selector interface {
	Next(ctx context.Context, n domain.Notification) (Provider, error)
}

// SelectorBuilder 供应商选择器构造器
type SelectorBuilder interface {
	Build() (Selector, error)
}
