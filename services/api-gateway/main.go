package main

import (
	"context"
	"log"      //создание логов
	"net/http" // работа с созданием сервера
	"os"
	"os/signal"
	"ride-sharing/shared/env" //модуль в котором переменные
	"ride-sharing/shared/messaging"
	"ride-sharing/shared/tracing"
	"syscall"
	"time"
)

var (
	httpAddr    = env.GetString("HTTP_ADDR", ":8081") // читаем переменну, и если ее нет то случаем порт
	rabbitMqURI = env.GetString("RABBITMQ_URI", "amqp://guest:guest@rabbitmq:5672/")

// Вы получаете значение переменной окружения "RABBITMQ_URI".
// Если переменная не установлена, то используется значение по умолчанию:
// "amqp://guest:guest@rabbitmq:5672/".
)

func main() {
	log.Println("Starting API Gateway")
	//запускаем сервер и на все запросы отвечаем хэло

	// Initialize Tracing
	tracerCfg := tracing.Config{
		ServiceName:    "api-gateway",
		Environment:    env.GetString("ENVIRONMENT", "development"),
		JaegerEndpoint: env.GetString("JAEGER_ENDPOINT", "http://jaeger:14268/api/traces"),
	}

	sh, err := tracing.InitTracer(tracerCfg)
	if err != nil {
		log.Fatalf("Failed to initialize the tracer: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer sh(ctx)

	mux := http.NewServeMux()
	// RabbitMQ connection
	rabbitmq, err := messaging.NewRabbitMQ(rabbitMqURI)
	if err != nil {
		log.Fatal(err)
	}
	defer rabbitmq.Close()

	log.Println("Starting RabbitMQ connection")                                                               //вместо хттп встроенный мультиплексер
	mux.Handle("POST /trip/preview", tracing.WrapHandlerFunc(enableCORS(handleTripPreview), "/trip/preview")) // подключаем корс//mux.HandleFunc(" POST /trip/preview", handleTripPreview) //  func(w http.ResponseWriter, r *http.Request) убрали это с кода и создали файл в это директории http
	mux.Handle("POST /trip/start", tracing.WrapHandlerFunc(enableCORS(handleTripStart), "/trip/start"))       // начало путеш
	// веб сокет для водителя и для челикса
	mux.Handle("/ws/drivers", tracing.WrapHandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleDriversWebSocket(w, r, rabbitmq)
	}, "/ws/drivers"))
	mux.Handle("/ws/riders", tracing.WrapHandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleRidersWebSocket(w, r, rabbitmq)
	}, "/ws/riders"))
	mux.Handle("/webhook/stripe", tracing.WrapHandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleStripeWebhook(w, r, rabbitmq)
	}, "/webhook/stripe"))
	//{
	//	w.WriteHeader(http.StatusOK)
	//	w.Write([]byte("Hello from API Gateway"))
	//})

	// http.ListenAndServe(httpAddr, nil) вариант до использования мультиплексера
	server := &http.Server{ //создание сервера с помощью мультиплексера
		Addr:    httpAddr,
		Handler: mux,
	}
	//
	serverErrors := make(chan error, 1)
	// Создаём канал для ошибок сервера с буфером 1 — здесь будут сигналы о ошибках сервера.

	go func() {
		log.Printf("Server listening on %s", server.Addr)
		serverErrors <- server.ListenAndServe()
		// Запускаем сервер в отдельной горутине и передаём в канал результат работы ListenAndServe.
		// Этот метод блокируется и возвращает ошибку, когда сервер прекращает работу.
	}()

	shutdown := make(chan os.Signal, 1)
	// Канал для получения системных сигналов, например, для остановки сервера.

	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)
	// Устанавливаем оповещения: канал shutdown будет получать сигналы прерывания (Ctrl+C — os.Interrupt)
	// и сигналы завершения из ОС (SIGTERM).

	select {
	case err := <-serverErrors:
		log.Printf("Error starting server: %v", err)
		// Если сервер остановился с ошибкой (например, порт уже занят),
		// выведем её в лог.

	case sig := <-shutdown:
		log.Printf("Server is shutting down due to %v signal", sig)
		// Если получили сигнал завершения от ОС, начинаем корректное выключение сервера.

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		// Создаём контекст с тайм-аутом 10 секунд, чтобы сервер успел
		// корректно завершить текущие запросы.

		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Could not stop server gracefully: %v", err)
			server.Close()
			// Если graceful shutdown (корректное завершение) не удался,
			// принудительно закрываем сервер.
		}
	}
}
