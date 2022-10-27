package queue

type QueueCB func([]byte) error

//定义任务队列的处理逻辑
type IFQueue interface {
	Close()
	Init() error
	SetQueueIndex(idx int)
	Clone(queue string) IFQueue
	Publish(data interface{}) error
	Consumer(handle QueueCB) error
	AsyncConsumer(conCurrency int, handle QueueCB) error
}
