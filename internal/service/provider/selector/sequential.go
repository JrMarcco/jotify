package selector

import (
	"context"
	"fmt"

	"github.com/JrMarcco/jotify/internal/domain"
	"github.com/JrMarcco/jotify/internal/errs"
	"github.com/JrMarcco/jotify/internal/service/provider"
)

var _ provider.Selector = (*SeqSelector)(nil)

type SeqSelector struct {
	index     int
	providers []provider.Provider
}

func (ss *SeqSelector) Next(ctx context.Context, n domain.Notification) (provider.Provider, error) {
	if len(ss.providers) == ss.index {
		return nil, fmt.Errorf("%w", errs.ErrNoAvailableProvider)
	}

	p := ss.providers[ss.index]
	ss.index++
	return p, nil
}

var _ provider.SelectorBuilder = (*SeqSelectorBuilder)(nil)

type SeqSelectorBuilder struct {
	providers []provider.Provider
}

func (ssb *SeqSelectorBuilder) Build() provider.SelectorBuilder {
	return &SeqSelectorBuilder{
		providers: ssb.providers,
	}
}

func NewSeqSelectorBuilder(providers []provider.Provider) *SeqSelectorBuilder {
	return &SeqSelectorBuilder{
		providers: providers,
	}
}
