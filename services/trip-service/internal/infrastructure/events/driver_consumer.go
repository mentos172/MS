package events
//Этот код реализует потребителя сообщений на роутинг-кейсе "ответ водителя" (принятие или отказ от поездки) 
// через RabbitMQ, и реагирует на эти события.
import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"ride-sharing/services/trip-service/internal/domain"
	"ride-sharing/shared/contracts"
	"ride-sharing/shared/messaging"
	pbd "ride-sharing/shared/proto/driver"

	"github.com/rabbitmq/amqp091-go"
)
//driverConsumer — структура, которая содержит подключение к RabbitMQ 
// и сервис для работы с поездками (domain.TripService).
type driverConsumer struct {
	rabbitmq *messaging.RabbitMQ
	service  domain.TripService
}

func NewDriverConsumer(rabbitmq *messaging.RabbitMQ, service domain.TripService) *driverConsumer {
	return &driverConsumer{
		rabbitmq: rabbitmq,
		service:  service,
	}
}
//Listen() — запускает цикл прослушивания сообщений из очереди messaging.DriverTripResponseQueue.
func (c *driverConsumer) Listen() error {
	return c.rabbitmq.ConsumeMessages(messaging.DriverTripResponseQueue, func(ctx context.Context, msg amqp091.Delivery) error {
		//Распарсивает из байтов JSON в структуру contracts.AmqpMessage.
        //Распарсивает Data в messaging.DriverTripResponseData.
        //В зависимости от RoutingKey (тип реакции драйвера) — вызывает нужный обработчик (например, handleTripAccepted).
		var message contracts.AmqpMessage
		if err := json.Unmarshal(msg.Body, &message); err != nil {
			log.Printf("Failed to unmarshal message: %v", err)
			return err
		}

		var payload messaging.DriverTripResponseData
		if err := json.Unmarshal(message.Data, &payload); err != nil {
			log.Printf("Failed to unmarshal message: %v", err)
			return err
		}

		log.Printf("driver response received message: %+v", payload)

		switch msg.RoutingKey {
		case contracts.DriverCmdTripAccept:
			if err := c.handleTripAccepted(ctx, payload.TripID, payload.Driver); err != nil {
				log.Printf("Failed to handle the trip accept: %v", err)
				return err
			}
		case contracts.DriverCmdTripDecline:
			if err := c.handleTripDeclined(ctx, payload.TripID, payload.RiderID); err != nil {
				log.Printf("Failed to handle the trip decline: %v", err)
				return err
			}
			return nil
		}
		log.Printf("unknown trip event: %+v", payload)

		return nil
	})
}

// если водитель отклонил мы пробуем найти другого водилу
func (c *driverConsumer) handleTripDeclined(ctx context.Context, tripID, riderID string) error {
	// When a driver declines, we should try to find another driver
	
	trip, err := c.service.GetTripByID(ctx, tripID)
	if err != nil {
		return err
	}

	newPayload := messaging.TripEventData{
		Trip: trip.ToProto(),
	}

	marshalledPayload, err := json.Marshal(newPayload)
	if err != nil {
		return err
	}

	if err := c.rabbitmq.PublishMessage(ctx, contracts.TripEventDriverNotInterested,
		contracts.AmqpMessage{
			OwnerID: riderID,
			Data:    marshalledPayload,
		},
	); err != nil {
		return err
	}

	return nil
}

func (c *driverConsumer) handleTripAccepted(ctx context.Context, tripID string, driver *pbd.Driver) error {
	// 1. Fetch the first
	//Взять текущую информацию о поездке по tripID.
	trip, err := c.service.GetTripByID(ctx, tripID)
	if err != nil {
		return err
	}

	if trip == nil {
		return fmt.Errorf("Trip was not found %s", tripID)
	}

	// 2. Update the trip
	//Обновить статус поездки на "accepted" и привязать водителя (driver).
	if err := c.service.UpdateTrip(ctx, tripID, "accepted", driver); err != nil {
		log.Printf("Failed to update the trip: %v", err)
		return err
	}

	trip, err = c.service.GetTripByID(ctx, tripID)
	if err != nil {
		return err
	}

	// 3. Driver has been assigned -> publish this event to RB
	//Еще раз получить обновленную информацию о поездке.
	marshalledTrip, err := json.Marshal(trip)
	if err != nil {
		return err
	}

	// Notify the rider that a driver has been assigned
	//Опубликовать событие о подтверждении, чтобы уведомить заказчика (рейтера).
	if err := c.rabbitmq.PublishMessage(ctx, contracts.TripEventDriverAssigned, contracts.AmqpMessage{
		OwnerID: trip.UserID,
		Data: marshalledTrip,
	}); err != nil {
		return err
	}

	// TODO: Notify the payment service to start a payment link

	return nil
}