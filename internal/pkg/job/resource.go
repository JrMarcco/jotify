package job

import (
	"context"
	"sync"

	"github.com/JrMarcco/jotify/internal/errs"
)

// ResourceSemaphore 信号量，用来控制抢占资源
type ResourceSemaphore interface {
	Acquire(ctx context.Context) error
	Release(ctx context.Context) error
}

var _ ResourceSemaphore = (*MaxCntResourceSemaphore)(nil)

type MaxCntResourceSemaphore struct {
	mu sync.Mutex

	maxCnt  int
	currCnt int
}

func (s *MaxCntResourceSemaphore) Acquire(_ context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.currCnt >= s.maxCnt {
		return errs.ErrAcquireExceedLimit
	}
	s.currCnt++
	return nil
}

func (s *MaxCntResourceSemaphore) Release(_ context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.currCnt--
	return nil
}

func (s *MaxCntResourceSemaphore) UpdateMaxCnt(maxCnt int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.maxCnt = maxCnt
}

func NewMaxCntResourceSemaphore(maxCnt int) *MaxCntResourceSemaphore {
	return &MaxCntResourceSemaphore{
		maxCnt:  maxCnt,
		currCnt: 0,
	}
}
