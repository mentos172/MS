package messaging
// нужно для того чтобы в каждом сервисе не перепрописывать подключение рабитмк
import (
	"context"
	"log"
	"fmt"
	"encoding/json"
	"ride-sharing/shared/contracts"
	amqp "github.com/rabbitmq/amqp091-go"// импорт клиентской библеотеки
)
// переменная для обменника можно и без нее
const (
	TripExchange = "trip"
)
// делаем структуру которая, которая содержит соединение с брокером
type RabbitMQ struct {
	conn    *amqp.Connection// указатель на обьект подключения
		Channel *amqp.Channel // подключение канала для отправки сообщений
}
// Функция для создания нового клиента RabbitMQ с соединением по указанному URI
func NewRabbitMQ(uri string) (*RabbitMQ, error) {
	conn, err := amqp.Dial(uri) // Пытаемся установить соединение с RabbitMQ по URI
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %v", err)
	}
	// сам канал и его закрытие в случае ошибки
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create channel: %v", err)
	}
// Создаем экземпляр RabbitMQ со установленным соединением
	rmq := &RabbitMQ{
		conn:    conn, // присваиваем соединение полю conn
		Channel: ch, // нужно для закрытия
	}
// если ошибка по передаче данных то закрыть 
	if err := rmq.setupExchangesAndQueues(); err != nil {
		// Clean up if setup fails
		rmq.Close()
		return nil, fmt.Errorf("failed to setup exchanges and queues: %v", err)
	}
//Возвращаем созданный объект RabbitMQ и nil ошибку
	return rmq, nil
}
// получаем сообщение и задержку или таймаут(контекст)
type MessageHandler func(context.Context, amqp.Delivery) error
//Метод для объекта RabbitMQ.
//Принимает:
//queueName — название очереди для прослушки.
//handler — функцию-обработчик для каждого сообщения.
//Возвращает ошибку, если возникла проблема.
func (r *RabbitMQ) ConsumeMessages(queueName string, handler MessageHandler) error {
	// Set prefetch count to 1 for fair dispatch
	// This tells RabbitMQ not to give more than one message to a service at a time.
	// The worker will only get the next message after it has acknowledged the previous one.
	err := r.Channel.Qos(
		1,     // prefetchCount: Limit to 1 unacknowledged message per consumer
		0,     // prefetchSize: No specific limit on message size
		false, // global: Apply prefetchCount to each consumer individually
	)
	if err != nil {
		return fmt.Errorf("failed to set QoS: %v", err)
	}
	msgs, err := r.Channel.Consume( //хапрос на получение сообщений
		queueName, // queue
		"",        // consumer
		true,      // auto-ack
		false,     // exclusive
		false,     // no-local
		false,     // no-wait
		nil,       // args
	)
	//Вызывается Consume, который начинает слушать указанную очередь.
//Возвращается канал msgs, через который поступают сообщения.
//Если ошибка — метод возвращает её.
	if err != nil {
		return err
	}
//создвется базовый контекст
	ctx := context.Background()
// запуск прослушивания горутины
//В отдельной горутине происходит бесконечный цикл (for msg := range msgs).
//Каждое сообщение обрабатывается вызовом handler.
//Если в обработчике возникла ошибка, программа завершится с фатальным логом (log.Fatalf).
	go func() {
		for msg := range msgs {
			log.Printf("Received a message: %s", msg.Body)
// контроль сообщений
			if err := handler(ctx, msg); err != nil {
				log.Printf("ERROR: Failed to handle message: %v. Message body: %s", err, msg.Body)
				// Nack the message. Set requeue to false to avoid immediate redelivery loops.
				// Consider a dead-letter exchange (DLQ) or a more sophisticated retry mechanism for production.
				if nackErr := msg.Nack(false, false); nackErr != nil {
					log.Printf("ERROR: Failed to Nack message: %v", nackErr)
				}
				
				// Continue to the next message
				continue
			}

			// Only Ack if the handler succeeds
			if ackErr := msg.Ack(false); ackErr != nil {
				log.Printf("ERROR: Failed to Ack message: %v. Message body: %s", ackErr, msg.Body)
			}
		}
	}()

	return nil
}
// отправка как раз таки самого сообщения
func (r *RabbitMQ) PublishMessage(ctx context.Context, routingKey string, message contracts.AmqpMessage) error {
	log.Printf("Publishing message with routing key: %s", routingKey)
	jsonMsg, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %v", err)
	}
	// асинхронная пуьликация
	return r.Channel.PublishWithContext(ctx,
		//"",      // exchange название обменника пустое поле
		//"hello", // routing key имя очереди куда попадет сообщение
		//false,   // mandatory
		//false,   // immediate
		TripExchange, // exchange
		routingKey,   // routing key
		false,        // mandatory
		false,        // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        jsonMsg,
			DeliveryMode: amqp.Persistent, //для сохрания
		})// структура указывающая содержимое сообщения
}
// насстройка очереди
//Публикует сообщение в обменник TripExchange.
//Использует PublishWithContext, что позволяет управлять тайм-аутами и отменами.
//В теле сообщения — переданный message.
//Устанавливает DeliveryMode: amqp.Persistent — сообщение сохранится даже после перезапуска RabbitMQ.
//Журналит, какой routing key используют.
func (r *RabbitMQ) setupExchangesAndQueues() error {
	//_, err := r.Channel.QueueDeclare(
	//	"hello", // name
	//	true,   // durable для сохраниения сооб
	//	false,   // delete when unused
	//	false,   // exclusive
	//	false,   // no-wait
	//	nil,     // arguments
	//)
	err := r.Channel.ExchangeDeclare(
		TripExchange, // name
		"topic",      // type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare exchange: %s: %v", TripExchange, err)
	}
// обьявление и связывание очередей
//Создает обменник TripExchange типа "topic" — это шаблонный обменник, позволяющий маршрутизировать сообщения по сложным правилам.
//Через declareAndBindQueue создает очередь и связывает ее с этим обменником по указанным routing key (TripEventCreated, TripEventDriverNotInterested).
	if err := r.declareAndBindQueue(
		FindAvailableDriversQueue,
		[]string{
			contracts.TripEventCreated, contracts.TripEventDriverNotInterested,
		},
		TripExchange,
	); err != nil {
		return err
	}
	//r.declareAndBindQueue — функция, которая, скорее всего, объявляет очередь и связывает её с обменником (exchange).
//DriverCmdTripRequestQueue — название самой очереди, которую нужно создать.
//[]string{contracts.DriverCmdTripRequest} — список маршрутов (routing keys), на которые очередь будет подписана.
//TripTrade — название обменника (exchange), с которым связывается очередь.

if err := r.declareAndBindQueue(
		DriverCmdTripRequestQueue,
		[]string{contracts.DriverCmdTripRequest},
		TripExchange,
	); err != nil {
		return err
	}
if err := r.declareAndBindQueue(
		DriverTripResponseQueue,
		// указаны ключи которые попадут сюда сообщения
		[]string{contracts.DriverCmdTripAccept, contracts.DriverCmdTripDecline},
		TripExchange,
	); err != nil {
		return err
	}

	if err := r.declareAndBindQueue(
		NotifyDriverNoDriversFoundQueue,
		[]string{contracts.TripEventNoDriversFound},
		TripExchange,
	); err != nil {
		return err
	}
	//declareAndBindQueue — функция, которая отвечает за:
//Создание очереди с именем NotifyDriverAssignQueue.
//Связывание этой очереди с обменником TripExchange по 
//маршрутизаторам contracts.TripEventDriverAssigned.
	if err := r.declareAndBindQueue(
		NotifyDriverAssignQueue,
		[]string{contracts.TripEventDriverAssigned},
		TripExchange,
	); err != nil {
		return err
	}
	return nil
}
//Создает очередь с именем queueName.
//Для каждого типа сообщения (messageTypes) связывает очередь с обменником по маршрутам (routing keys).
//Использует цикл — для каждой темы (msg) вызывает QueueBind, связывая очередь с обменником.
func (r *RabbitMQ) declareAndBindQueue(queueName string, messageTypes []string, exchange string) error {
	q, err := r.Channel.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)

	if err != nil {
		log.Fatal(err)
	}
for _, msg := range messageTypes {
		if err := r.Channel.QueueBind(
			q.Name,   // queue name
			msg,      // routing key
			exchange, // exchange
			false,
			nil,
		); err != nil {
			return fmt.Errorf("failed to bind queue to %s: %v", queueName, err)
		}
	}

	return nil
}
// Метод для закрытия соединения с RabbitMQ
func (r *RabbitMQ) Close() {
	if r.conn != nil {  // Проверяем, что соединение не равно nil
		r.conn.Close() //закрываем
	}
	if r.Channel != nil {
		r.Channel.Close()
	}
}