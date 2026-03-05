package events

import (
	"context"
	"encoding/json"
	"ride-sharing/services/trip-service/internal/domain"
	"ride-sharing/shared/contracts"
	"ride-sharing/shared/messaging"
)

type TripEventPublisher struct {
	rabbitmq *messaging.RabbitMQ // для публикации
}
//NewTripEventPublisher — функция-конструктор, создающая новый экземпляр TripEventPublisher.
//rabbitmq *messaging.RabbitMQ — параметр, передаваемый извне, чтобы связать этот обработчик с внешней системой сообщений.
//Возвращает указатель на новый TripEventPublisher, где в поле rabbitmq вставляется переданный объект.

func NewTripEventPublisher(rabbitmq *messaging.RabbitMQ) *TripEventPublisher {
	return &TripEventPublisher{
		rabbitmq: rabbitmq,
	}
}
// передача самого сообщения

func (p *TripEventPublisher) PublishTripCreated(ctx context.Context, trip *domain.TripModel) error {
	payload := messaging.TripEventData{
		Trip: trip.ToProto(),
	}

	tripEventJSON, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	return p.rabbitmq.PublishMessage(ctx, contracts.TripEventCreated, contracts.AmqpMessage{
		OwnerID: trip.UserID,
		Data:    tripEventJSON,
	})
}