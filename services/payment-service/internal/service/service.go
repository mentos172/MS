package service

import (
	"context"
	"fmt"
	"time"

	"ride-sharing/services/payment-service/internal/domain"
	"ride-sharing/services/payment-service/pkg/types"

	"github.com/google/uuid"
)

type paymentService struct {
	paymentProcessor domain.PaymentProcessor
}

// NewPaymentService creates a new instance of the payment service
func NewPaymentService(paymentProcessor domain.PaymentProcessor) domain.Service {
	return &paymentService{
		paymentProcessor: paymentProcessor,
	}
}

// CreatePaymentSession creates a new payment session for a trip
func (s *paymentService) CreatePaymentSession(
	ctx context.Context,
	tripID string,
	userID string,
	driverID string,
	amount int64,
	currency string,
) (*types.PaymentIntent, error) {
	metadata := map[string]string{
		"trip_id":   tripID,
		"user_id":   userID,
		"driver_id": driverID,
	}
	//Создает metadata с идентификаторами поездки, пользователя и водителя.
	//Передает эти метаданные вместе с суммой и валютой в paymentProcessor.CreatePaymentSession.
	//Получает sessionID — идентификатор платежной сессии (например, в Stripe).
	//Создает структуру types.PaymentIntent, наполняя её данными:
	//ID — уникальный идентификатор платежа (генерируется через uuid.New()).
	//Ссылки на поездку, пользователя, водителя, сумму, валюту.
	//StripeSessionID — ID сессии в платежной системе.
	//CreatedAt — время создания.
	//Возвращает созданный PaymentIntent или ошибку.
	sessionID, err := s.paymentProcessor.CreatePaymentSession(ctx, amount, currency, metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to create payment session: %w", err)
	}

	paymentIntent := &types.PaymentIntent{
		ID:              uuid.New().String(),
		TripID:          tripID,
		UserID:          userID,
		DriverID:        driverID,
		Amount:          amount,
		Currency:        currency,
		StripeSessionID: sessionID,
		CreatedAt:       time.Now(),
	}

	return paymentIntent, nil
}
