package domain

type Quota struct {
	BizId   uint64
	Quota   int32
	Channel Channel
}
