package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"ride-sharing/services/payment-service/pkg/types"
	"ride-sharing/shared/env"
	"ride-sharing/shared/messaging"
)

var GrpcAddr = env.GetString("GRPC_ADDR", ":9004")

//Чтение переменной окружения RABBITMQ_URI — URI для подключения к RabbitMQ.
//Создаётся context.Context с функцией отмены cancel() для корректного завершения работы всего сервиса.
//Горутинa слушает системные сигналы (SIGINT, SIGTERM) и вызывает cancel(), что инициирует завершение сервиса.
func main() {
	rabbitMqURI := env.GetString("RABBITMQ_URI", "amqp://guest:guest@rabbitmq:5672/")

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh
		cancel()
	}()

	appURL := env.GetString("APP_URL", "http://localhost:3000")

	// Stripe config
	//Переменная appURL — базовый URL приложения.
    //Конфигурация Stripe:
    //StripeSecretKey — секретный ключ API.
    //URL-ы для успешного завершения и отмены платежа, с умолчанием, основанным на appURL.
	stripeCfg := &types.PaymentConfig{
		StripeSecretKey: env.GetString("STRIPE_SECRET_KEY", ""),
		SuccessURL:      env.GetString("STRIPE_SUCCESS_URL", appURL+"?payment=success"),
		CancelURL:       env.GetString("STRIPE_CANCEL_URL", appURL+"?payment=cancel"),
	}

	if stripeCfg.StripeSecretKey == "" {
		log.Fatalf("STRIPE_SECRET_KEY is not set")
		return
	}

	// RabbitMQ connection
	//Создается соединение с RabbitMQ через функцию messaging.NewRabbitMQ.
    //В случае ошибки — программа завершается.
//defer rabbitmq.Close() — при завершении программы закрывает соединение.
	rabbitmq, err := messaging.NewRabbitMQ(rabbitMqURI)
	if err != nil {
		log.Fatal(err)
	}
	defer rabbitmq.Close()

	log.Println("Starting RabbitMQ connection")

	// Wait for shutdown signal
	<-ctx.Done()
	log.Println("Shutting down payment service...")
}