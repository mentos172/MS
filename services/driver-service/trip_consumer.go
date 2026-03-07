package main

import (
	"context"
	"log"
	"ride-sharing/shared/messaging"
    "encoding/json"
    "ride-sharing/shared/contracts"
	"github.com/rabbitmq/amqp091-go"
	"math/rand"
)
//Структура хранит ссылку на вашу реализацию RabbitMQ.
//Конструктор создает новую tripConsumer.
type tripConsumer struct {
	rabbitmq *messaging.RabbitMQ
	service *Service
}

func NewTripConsumer(rabbitmq *messaging.RabbitMQ, service *Service) *tripConsumer {
	return &tripConsumer{
		rabbitmq: rabbitmq,
		service: service,
	}
}
//Запускает чтение сообщений из очереди "hello" (название очереди).
//Передает анонимный обработчик, который просто логирует сообщение и возвращает nil — успех.
func (c *tripConsumer) Listen() error {
	return c.rabbitmq.ConsumeMessages(messaging.FindAvailableDriversQueue, func(ctx context.Context, msg amqp091.Delivery) error {
	var tripEvent contracts.AmqpMessage
		if err := json.Unmarshal(msg.Body, &tripEvent); err != nil {
			log.Printf("Failed to unmarshal message: %v", err)
			return err
		}

		var payload messaging.TripEventData
		if err := json.Unmarshal(tripEvent.Data, &payload); err != nil {
			log.Printf("Failed to unmarshal message: %v", err)
			return err
		}

		log.Printf("driver received message: %+v", payload)
		switch msg.RoutingKey {
		case contracts.TripEventCreated, contracts.TripEventDriverNotInterested:
			return c.handleFindAndNotifyDrivers(ctx, payload)
		}

		log.Printf("unknown trip event: %+v", payload)
		return nil
	})
	}

	func (c *tripConsumer) handleFindAndNotifyDrivers(ctx context.Context, payload messaging.TripEventData) error {
	// поиск водителя
		suitableIDs := c.service.FindAvailableDrivers(payload.Trip.SelectedFare.PackageSlug)
//логируе число подходящих водителей
	log.Printf("Found suitable drivers %v", len(suitableIDs))
//проверка есть ли водитли
	if len(suitableIDs) == 0 {
		// Notify the driver that no drivers are available
		if err := c.rabbitmq.PublishMessage(ctx, contracts.TripEventNoDriversFound, contracts.AmqpMessage{
			OwnerID: payload.Trip.UserID,
		}); err != nil {
			log.Printf("Failed to publish message to exchange: %v", err)
			return err
		}

		return nil
	}
// выбор random подходящего
		// Get a random index from the matching drivers
	randomIndex := rand.Intn(len(suitableIDs))

	suitableDriverID := suitableIDs[randomIndex]
//Оборачиваем payload в JSON для отправки в сообщение.
	marshalledEvent, err := json.Marshal(payload)
	if err != nil {
		return err
	}
// уведомляем выбранного водилу
	// Notify the driver about a potential trip
	if err := c.rabbitmq.PublishMessage(ctx, contracts.DriverCmdTripRequest, contracts.AmqpMessage{
		OwnerID: suitableDriverID,
		Data: marshalledEvent,
	}); err != nil {
		log.Printf("Failed to publish message to exchange: %v", err)
		return err
	}

	return nil
}
