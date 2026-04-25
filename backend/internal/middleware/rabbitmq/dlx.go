package rabbitmq

import (
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	DLXExchange   = "dlx.events"
	MaxRetryCount = 3
)

// DeclareDLX 声明死信交换机和对应的死信队列
func DeclareDLX(ch *amqp.Channel, queueName string) error {
	if ch == nil {
		return nil
	}
	if err := ch.ExchangeDeclare(
		DLXExchange, "topic", true, false, false, false, nil,
	); err != nil {
		return err
	}
	dlxQueue := queueName + ".dlx"
	_, err := ch.QueueDeclare(
		dlxQueue, true, false, false, false, nil,
	)
	if err != nil {
		return err
	}
	if err := ch.QueueBind(dlxQueue, "#", DLXExchange, false, nil); err != nil {
		return err
	}
	log.Printf("DLX ready: exchange=%s queue=%s", DLXExchange, dlxQueue)
	return nil
}

// GetRetryCount 从 AMQP x-death header 中提取当前消息已被重试的次数
func GetRetryCount(d amqp.Delivery) int {
	deaths, ok := d.Headers["x-death"].([]interface{})
	if !ok || len(deaths) == 0 {
		return 0
	}
	death, ok := deaths[0].(amqp.Table)
	if !ok {
		return 0
	}
	count, ok := death["count"].(int64)
	if !ok {
		return 0
	}
	return int(count)
}
