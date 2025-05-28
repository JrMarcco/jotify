package sender

// Sender 消息发送接口（实际的发送逻辑）
//
// 相比之下 SendService 处理的是发送前的准备工作。
// 例如 Notification 记录入库，配额
type Sender interface{}

var _ Sender = (*DefaultSender)(nil)

type DefaultSender struct {
}
