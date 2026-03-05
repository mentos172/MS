package messaging

import (
	"encoding/json"
	"log"

	"ride-sharing/shared/contracts"
)

type QueueConsumer struct {
	rb        *RabbitMQ // указатель на объект RabbitMQ, который содержит канал (Channel) и, вероятно, соединение.
	connMgr   *ConnectionManager //connMgr: менеджер WebSocket соединений, отвечает за отправку сообщений клиентам.
	queueName string //queueName: имя очереди, которую этот потребитель слушает.
}
//Создаёт новый экземпляр QueueConsumer с переданными параметрами.
func NewQueueConsumer(rb *RabbitMQ, connMgr *ConnectionManager, queueName string) *QueueConsumer {
	return &QueueConsumer{
		rb:        rb,
		connMgr:   connMgr,
		queueName: queueName,
	}
}
//Вызов Consume из канала RabbitMQ запускает подписку на очередь.
func (qc *QueueConsumer) Start() error {
	msgs, err := qc.rb.Channel.Consume(
		qc.queueName,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}
//Создаётся горутина, которая обрабатывает входящие сообщения.
//Каждое сообщение (msg) — это amqp.Delivery.
//Его тело (msg.Body) декодируется в структуру contracts.AmqpMessage.
//Если не получилось — логируется и продолжается цикл.
	go func() {
		for msg := range msgs {
			var msgBody contracts.AmqpMessage
			if err := json.Unmarshal(msg.Body, &msgBody); err != nil {
				log.Println("Failed to unmarshal message:", err)
				continue
			}
//Вытаскивает OwnerID — идентификатор пользователя, которому предназначено сообщение.
//Потом, если есть данные (msgBody.Data), их десериализует в переменную payload.
//Если возникла ошибка — логируем и переходим к следующему сообщению.
			userID := msgBody.OwnerID

			var payload any
			if msgBody.Data != nil {
				if err := json.Unmarshal(msgBody.Data, &payload); err != nil {
					log.Println("Failed to unmarshal payload:", err)
					continue
				}
			}
//Создаёт сообщение WSMessage, где:
//Type — тип сообщения (использует RoutingKey),
//Data — десериализованный payload.
//Передаёт сообщение через менеджер соединений SendMessage(userID, clientMsg).
//Если есть ошибка — логируем.
			clientMsg := contracts.WSMessage{
				Type: msg.RoutingKey,
				Data: payload,
			}

			if err := qc.connMgr.SendMessage(userID, clientMsg); err != nil {
				log.Printf("Failed to send message to user %s: %v", userID, err)
			}
		}
	}()

	return nil
}