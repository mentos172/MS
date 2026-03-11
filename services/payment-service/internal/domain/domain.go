package domain

import (
	"context"

	"ride-sharing/services/payment-service/pkg/types"
)

// Предоставляет метод CreatePaymentSession, который:
// принимает контекст ctx — для отмены и дедлайнов.
// параметры:
// tripID — идентификатор поездки.
// userID — пользователь, платящий.
// driverID — водитель, возможно, для уведомлений или учета.
// amount — сумма платежа в минимальных единицах (например, копейки).
// currency — валюта (например, "RUB", "USD").
// возвращает:
// указатель на types.PaymentIntent — provavelmente, структура, содержащая детали платежа.
// ошибку.
type Service interface {
	CreatePaymentSession(ctx context.Context, tripID, userID, driverID string, amount int64, currency string) (*types.PaymentIntent, error)
}

// Предоставляет внутренние методы работы с платежами у конкретного провайдера (Stripe, Яндекс.Касса, PayPal и др.).
// Методы:
// CreatePaymentSession — создает платежную сессию с минимальной информацией:
// amount, currency, metadata.
// возвращает sessionID (или его аналог) и ошибку.
// GetSessionStatus — проверяет статус сессии по sessionID, возвращая статус вида PaymentStatus.
type PaymentProcessor interface {
	CreatePaymentSession(ctx context.Context, amount int64, currency string, metadata map[string]string) (string, error)
	//GetSessionStatus(ctx context.Context, sessionID string) (types.PaymentStatus, error)
}
