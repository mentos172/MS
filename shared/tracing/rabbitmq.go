package tracing

import (
	"context"
	"encoding/json"

	"ride-sharing/shared/contracts"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// amqpHeadersCarrier implements the TextMapCarrier interface for AMQP headers
type amqpHeadersCarrier amqp.Table

// Реализует интерфейс TextMapCarrier, который нужен OpenTelemetry для вставки и извлечения контекста из заголовков сообщений RabbitMQ (amqp.Table).
// Методы:
// Get: получает значение по ключу (возвращает строку).
// Set: устанавливает значение по ключу.
// Keys: возвращает список ключей.
// Это позволяет проложить трассировочные данные внутри заголовков сообщений RabbitMQ для передачи информации о трассировке между сервисами.
func (c amqpHeadersCarrier) Get(key string) string {
	if v, ok := c[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func (c amqpHeadersCarrier) Set(key string, value string) {
	c[key] = value
}

func (c amqpHeadersCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}

// TracedPublisher wraps the RabbitMQ publish function with tracing
//Эта функция оборачивает процесс отправки сообщений:

//Создает новый span (rabbitmq.publish) для отслеживания процесса отправки.
//Устанавливает атрибуты:
//messaging.destination: название exchange, куда публикуем.
//messaging.routing_key: routing key.
//Пытается разборить тело сообщения (msg.Body) в структуру contracts.AmqpMessage. Если удачно — добавляет OwnerID как атрибут, что полезно для бизнес-аналитики или отладки.
//Вставляет контекст трассировки в заголовки сообщения через Inject.
//Вызывает функцию publish, которая выполняет реальную отправку.
//При ошибке указывает её в статусе спана.

func TracedPublisher(ctx context.Context, exchange, routingKey string, msg amqp.Publishing, publish func(context.Context, string, string, amqp.Publishing) error) error {
	tracer := otel.GetTracerProvider().Tracer("rabbitmq")

	ctx, span := tracer.Start(ctx, "rabbitmq.publish",
		trace.WithAttributes(
			attribute.String("messaging.destination", exchange),
			attribute.String("messaging.routing_key", routingKey),
		),
	)
	defer span.End()

	// Try to extract and add message details to span (map[string]any if you don't know the type)
	var msgBody contracts.AmqpMessage
	if err := json.Unmarshal(msg.Body, &msgBody); err == nil {
		if msgBody.OwnerID != "" {
			span.SetAttributes(attribute.String("messaging.owner_id", msgBody.OwnerID))
		}
	}

	// Inject trace context into message headers
	if msg.Headers == nil {
		msg.Headers = make(amqp.Table)
	}
	carrier := amqpHeadersCarrier(msg.Headers)
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	msg.Headers = amqp.Table(carrier)

	if err := publish(ctx, exchange, routingKey, msg); err != nil {
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	return nil
}

// TracedConsumer wraps the RabbitMQ message handler with tracing

//Обертка для функции обработки принятых сообщений:

//Извлекает контекст из заголовков сообщения (delivery.Headers) через Extract.
//Создает новый span (rabbitmq.consume) с атрибутами:
//messaging.destination: exchange, из которого пришло сообщение.
//messaging.routing_key: routing key.
//Пытается разобрать содержимое delivery.Body так же, как и в TracedPublisher, чтобы добавить OwnerID.
//Запускает обработчик handler внутри контекста с трейсовой информацией.
//Если обработка завершилась с ошибкой, отмечает ее в статусе спана.

func TracedConsumer(delivery amqp.Delivery, handler func(context.Context, amqp.Delivery) error) error {
	// Extract trace context from message headers
	carrier := amqpHeadersCarrier(delivery.Headers)
	ctx := otel.GetTextMapPropagator().Extract(context.Background(), carrier)

	tracer := otel.GetTracerProvider().Tracer("rabbitmq")

	ctx, span := tracer.Start(ctx, "rabbitmq.consume",
		trace.WithAttributes(
			attribute.String("messaging.destination", delivery.Exchange),
			attribute.String("messaging.routing_key", delivery.RoutingKey),
		),
	)
	defer span.End()

	// Try to extract and add message details to span (map[string]any if you don't know the type)
	var msgBody contracts.AmqpMessage
	if err := json.Unmarshal(delivery.Body, &msgBody); err == nil {
		if msgBody.OwnerID != "" {
			span.SetAttributes(attribute.String("messaging.owner_id", msgBody.OwnerID))
		}
	}

	if err := handler(ctx, delivery); err != nil {
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	return nil
}
