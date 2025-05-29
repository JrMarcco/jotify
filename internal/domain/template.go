package domain

import (
	"fmt"

	"github.com/JrMarcco/jotify/internal/errs"
)

// AuditStatus 审批状态
type AuditStatus string

const (
	AuditStatusPending   AuditStatus = "pending"
	AuditStatusInPreview AuditStatus = "in_preview"
	AuditStatusApproved  AuditStatus = "approved"
	AuditStatusRejected  AuditStatus = "rejected"
)

func (a AuditStatus) String() string {
	return string(a)
}

func (a AuditStatus) IsPending() bool {
	return a == AuditStatusPending
}

func (a AuditStatus) IsInPreview() bool {
	return a == AuditStatusInPreview
}

func (a AuditStatus) IsApproved() bool {
	return a == AuditStatusApproved
}

func (a AuditStatus) IsRejected() bool {
	return a == AuditStatusRejected
}

func (a AuditStatus) Validate() bool {
	switch a {
	case AuditStatusPending, AuditStatusInPreview, AuditStatusApproved, AuditStatusRejected:
		return true
	}
	return false
}

// OwnerType 所有者类型（个人/组织）
type OwnerType string

const (
	OwnerTypePerson       OwnerType = "person"
	OwnerTypeOrganization OwnerType = "organization"
)

func (o OwnerType) String() string {
	return string(o)
}

func (o OwnerType) Validate() bool {
	return o == OwnerTypePerson || o == OwnerTypeOrganization
}

// BizType 业务类型
type BizType string

const (
	BizTypePromotion    BizType = "Promotion"
	BizTypeNotification BizType = "notification"
	BizTypeVerifyCode   BizType = "verify_code"
)

func (b BizType) String() string {
	return string(b)
}

func (b BizType) Validate() bool {
	return b == BizTypePromotion || b == BizTypeNotification || b == BizTypeVerifyCode
}

// ChannelTpl 渠道模板领域对象
type ChannelTpl struct {
	Id                 uint64
	OwnerId            uint64
	OwnerType          OwnerType
	Name               string
	Description        string
	Channel            Channel
	BizType            BizType
	ActivatedVersionId uint64
	CreateAt           int64
	UpdateAt           int64
	Versions           []ChannelTplVersion
}

func (ct ChannelTpl) Validate() error {
	if !ct.OwnerType.Validate() {
		return fmt.Errorf("%w invalid owner type", errs.ErrInvalidParam)
	}

	if !ct.Channel.Validate() {
		return fmt.Errorf("%w invalid channel", errs.ErrInvalidParam)
	}

	if !ct.BizType.Validate() {
		return fmt.Errorf("%w invalid biz type", errs.ErrInvalidParam)
	}

	if ct.OwnerId <= 0 {
		return fmt.Errorf("%w owner id should not be negative or zero", errs.ErrInvalidParam)
	}

	if ct.Name == "" {
		return fmt.Errorf("%w template name should not be empty", errs.ErrInvalidParam)
	}

	if ct.Description == "" {
		return fmt.Errorf("%w template description should not be empty", errs.ErrInvalidParam)
	}

	return nil
}

func (ct ChannelTpl) Published() bool {
	return ct.ActivatedVersionId > 0
}

func (ct ChannelTpl) ActivatedVersion() *ChannelTplVersion {
	if ct.ActivatedVersionId <= 0 {
		return nil
	}

	for _, v := range ct.Versions {
		if v.Id == ct.ActivatedVersionId {
			return &v
		}
	}
	return nil
}

func (ct ChannelTpl) GetVersion(versionId uint64) *ChannelTplVersion {
	for _, v := range ct.Versions {
		if v.Id == versionId {
			return &v
		}
	}
	return nil
}

func (ct ChannelTpl) GetProvidersByVersion(versionId uint64) []ChannelTplProvider {
	version := ct.GetVersion(versionId)
	if version == nil {
		return nil
	}

	return version.Providers
}

func (ct ChannelTpl) GetProvider(versionId uint64, providerId uint64) *ChannelTplProvider {
	version := ct.GetVersion(versionId)
	if version == nil {
		return nil
	}

	for _, p := range version.Providers {
		if p.Id == providerId {
			return &p
		}
	}
	return nil
}

func (ct ChannelTpl) HasApprovedVersion() bool {
	for _, v := range ct.Versions {
		if v.AuditStatus == AuditStatusApproved {
			return true
		}
	}
	return false
}

// ChannelTplVersion 渠道模板版本领域对象
type ChannelTplVersion struct {
	Id           uint64
	ChannelTplId uint64
	Name         string
	Signature    string
	Content      string
	Remark       string
	AuditId      uint64
	AuditorId    uint64
	AuditAt      int64
	AuditStatus  AuditStatus
	RejectReason string
	LastReviewAt int64
	CreateAt     int64
	UpdateAt     int64
	Providers    []ChannelTplProvider
}

// ChannelTplProvider 渠道模板供应商领域对象
type ChannelTplProvider struct {
	Id              uint64
	TplId           uint64
	TplVersionId    uint64
	ProviderId      uint64
	ProviderName    string
	ProviderChannel Channel
	RequestId       string
	ProviderTplId   string
	AuditStatus     AuditStatus
	RejectReason    string
	LastReviewAt    int64
	CreateAt        int64
	UpdateAt        int64
}
