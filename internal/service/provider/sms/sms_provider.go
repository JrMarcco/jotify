package sms

import (
	"context"
	"fmt"
	"strings"

	"github.com/JrMarcco/jotify/internal/domain"
	"github.com/JrMarcco/jotify/internal/errs"
	"github.com/JrMarcco/jotify/internal/repository"
	"github.com/JrMarcco/jotify/internal/service/provider/sms/client"
)

type Provider struct {
	name         string
	client       client.SmsClient
	tplRepo      repository.ChannelTplRepo
	providerRepo repository.ProviderRepo
}

func (p *Provider) Send(ctx context.Context, n domain.Notification) (domain.SendResp, error) {
	channelTpl, err := p.getTemplate(ctx, n.Template.Id)
	if err != nil {
		return domain.SendResp{}, fmt.Errorf("%w: %w", errs.ErrFailedToSendNotification, err)
	}

	activatedVersion := channelTpl.ActivatedVersion()
	if activatedVersion == nil {
		return domain.SendResp{}, fmt.Errorf("%w: no published templates found", errs.ErrFailedToSendNotification)
	}

	const first = 0
	providerTplId := activatedVersion.Providers[first].ProviderTplId
	resp, err := p.client.Send(client.SendReq{
		PhoneNumbers:   n.Receivers,
		SignName:       activatedVersion.Signature,
		TemplateId:     providerTplId,
		TemplateParams: n.Template.Params,
	})
	if err != nil {
		return domain.SendResp{}, fmt.Errorf("%w: %w", errs.ErrFailedToSendNotification, err)
	}

	for _, status := range resp.PhoneNumbers {
		if !strings.EqualFold(status.Code, "OK") {
			return domain.SendResp{}, fmt.Errorf("%w: Response Code = %s, Response Message = %s", errs.ErrFailedToSendNotification, status.Code, status.Message)
		}
	}

	return domain.SendResp{
		Result: domain.SendResult{
			NotificationId: n.Id,
			Status:         domain.SendStatusSuccess,
		},
	}, nil
}

func (p *Provider) getTemplate(ctx context.Context, templateId uint64) (domain.ChannelTpl, error) {
	// 获取渠道模板主体信息
	tpl, err := p.tplRepo.GetById(ctx, templateId)
	if err != nil {
		return domain.ChannelTpl{}, err
	}
	if tpl.Id == 0 {
		return domain.ChannelTpl{}, fmt.Errorf("%w: template id = %d", errs.ErrChannelTplNotFound, tpl.Id)
	}

	// 获取当前生效的模板版本
	version, err := p.tplRepo.GetVersionByVersionId(ctx, tpl.ActivatedVersionId)
	if err != nil {
		return domain.ChannelTpl{}, err
	}
	if version.AuditStatus != domain.AuditStatusApproved {
		return domain.ChannelTpl{}, fmt.Errorf("%w: template version id = %d", errs.ErrNotApprovedTplVersion, version.Id)
	}

	// 根据生效的版本信息获取供应商信息
	providers, err := p.providerRepo.GetByNameAndTplInfo(ctx, p.name, tpl.Id, version.Id, domain.ChannelSMS.String())
	if err != nil {
		return domain.ChannelTpl{}, err
	}
	if len(providers) == 0 {
		return domain.ChannelTpl{}, fmt.Errorf("%w: template id = %d, version id = %d", errs.ErrNoAvailableProvider, tpl.Id, version.Id)
	}

	version.Providers = providers
	tpl.Versions = []domain.ChannelTplVersion{version}
	return tpl, nil
}

func NewProvider(
	name string, client client.SmsClient, tplRepo repository.ChannelTplRepo, providerRepo repository.ProviderRepo,
) *Provider {
	return &Provider{
		name:    name,
		client:  client,
		tplRepo: tplRepo,
	}
}
