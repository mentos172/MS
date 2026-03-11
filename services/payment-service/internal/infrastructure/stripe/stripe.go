package stripe

//Этот код реализует клиент для взаимодействия со Stripe API и создаёт сессию оплаты.
//Такой клиент обычно использует для интеграции с платежной системой, чтобы инициировать оплату
import (
	"context"
	"fmt"
	"ride-sharing/services/payment-service/internal/domain"
	"ride-sharing/services/payment-service/pkg/types"

	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/checkout/session"
)

// храним секретный ключ, чтобы каждый метод мог обратиться к кнужным настройкам без повторной передачи
type stripeClient struct {
	config *types.PaymentConfig
}

// Устанавливает глобальный API-ключ Stripe через stripe.Key.
// Создаёт и возвращает указатель на stripeClient.
// После вызова все методы клиента смогут автоматически работать с авторизованным API.
func NewStripeClient(config *types.PaymentConfig) domain.PaymentProcessor {
	stripe.Key = config.StripeSecretKey

	return &stripeClient{
		config: config,
	}
}

func (s *stripeClient) CreatePaymentSession(ctx context.Context, amount int64, currency string, metadata map[string]string) (string, error) {

	//SuccessURL и CancelURL:

	//URL, куда пользователь перенаправляется после успешной оплаты или отмены.
	//Параметры устанавливаются через stripe.String(), чтобы указать указатель на строку.
	//Metadata:

	//Передаваемые дополнительные данные (например, ID заказа или пользователя).
	//LineItems: список товаров/услуг, платёж которых создаётся.

	//В вашем случае — один товар ("Ride Payment").
	//Важный нюанс: UnitAmount — цена в минимальных единицах валюты. Например, $10.00 → 1000 центов.
	//PriceData:

	//Задаёт детали товара: валюта, название, цена.
	//Quantity: количество товара; в вашем случае — 1.

	//Mode:

	//режим платёжной сессии.
	//stripe.CheckoutSessionModePayment — это константа, равная "payment".
	//В вашем коде: stripe.String(string(stripe.CheckoutSessionModePayment)).

	params := &stripe.CheckoutSessionParams{
		SuccessURL: stripe.String(s.config.SuccessURL),
		CancelURL:  stripe.String(s.config.CancelURL),
		Metadata:   metadata,
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency: stripe.String(currency),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name: stripe.String("Ride Payment"),
					},
					UnitAmount: stripe.Int64(amount),
				},
				Quantity: stripe.Int64(1),
			},
		},
		Mode: stripe.String(string(stripe.CheckoutSessionModePayment)),
	}
	//Стандартный вызов API Stripe, создающий новую checkout-сессию.
	result, err := session.New(params)
	if err != nil {
		return "", fmt.Errorf("failed to create a payment session on stripe: %w", err)
	}
	//Возвращается ID созданной сессии — его используют в frontend для перенаправления клиента на оплату.
	return result.ID, nil
}
