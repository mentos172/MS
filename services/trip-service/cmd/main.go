// запускает HTTP сервер для сервиса,
// связанного с поездками (trip-service) в системе
// / изменено из http yf grpc
package main

import (
	"context"
	"log"

	//"net/http"
	"net"
	"os"
	"os/signal"

	//h "ride-sharing/services/trip-service/internal/infrastructure/http"     // обработчик
	//"ride-sharing/services/trip-service/internal/infrastructure/repository" //слой доступа к данным
	//"ride-sharing/services/trip-service/internal/service"
	"syscall"
	//"time"
	"ride-sharing/services/trip-service/internal/infrastructure/events"
	"ride-sharing/services/trip-service/internal/infrastructure/grpc"
	"ride-sharing/services/trip-service/internal/infrastructure/repository"
	"ride-sharing/services/trip-service/internal/service"
	"ride-sharing/shared/env"
	"ride-sharing/shared/messaging"
	"ride-sharing/shared/tracing"

	grpcserver "google.golang.org/grpc"
)

var GrpcAddr = ":9093"

func main() {
	//ctx := context.Background() во 2 версии убирается
	//rabbitMqURI := env.GetString("RABBITMQ_URI", "amqp://user:user@rabbitmq:5672/")

	// Initialize Tracing
	tracerCfg := tracing.Config{
		ServiceName:    "trip-service",
		Environment:    env.GetString("ENVIRONMENT", "development"),
		JaegerEndpoint: env.GetString("JAEGER_ENDPOINT", "http://jaeger:14268/api/traces"),
	}

	///inmemRepo := repository.NewInmemRepository() //создание репозитория который хранит данные

	///svc := service.NewService(inmemRepo) // создаем экземпляр сервиса биз логики - он принемает репозиторий
	sh, err := tracing.InitTracer(tracerCfg)
	if err != nil {
		log.Fatalf("Failed to initialize the tracer: %v", err)
	}
	///mux := http.NewServeMux()
	//fare := &domain.RideFareModel{
	//	UserID: "42",   использовали в проверке
	//}
	///httphandler := h.HttpHandler{Service: svc} //инициализация  обработчика

	//t, err := svc.CreateTrip(ctx, fare) //вызов метода создания поездки
	//if err != nil {
	//	log.Println(err)
	//}
	///mux.HandleFunc("POST /preview", httphandler.HandleTripPreview)
	//log.Println(t) //выводим созданную поездку
	///server := &http.Server{
	///	Addr:    ":8083",
	///	Handler: mux,
	///}
	// keep the program running for now бесконечный мотоцикл
	//for {
	//	time.Sleep(time.Second)
	//}

	//Создаёт in-memory репозиторий (хранилище данных).
	//Создаёт сервис, обрабатывающий бизнес-логику поездок, который работает с репозиторием.
	//Создаёт HTTP обработчик (обертку вокруг сервиса).
	//Регистрирует маршрут /preview для HTTP запросов (хотя текущая регистрация с ошибкой — нужно исправить путь).
	//Запускает HTTP сервер на порту 8083.
	//В результате приложение принимает HTTP запросы на localhost:8083/preview, обрабатывает их с помощью сервиса поездок.
	//создание горутины для отключения сервера коменты в апи гэтвый мэин го там тоже самое
	///serverErrors := make(chan error, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	defer sh(ctx)

	rabbitMqURI := env.GetString("RABBITMQ_URI", "amqp://guest:guest@rabbitmq:5672/")

	inmemRepo := repository.NewInmemRepository()
	svc := service.NewService(inmemRepo)

	go func() {
		///log.Printf("Server listening on %s", server.Addr)
		///serverErrors <- server.ListenAndServe()
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh
		cancel()
	}()

	///shutdown := make(chan os.Signal, 1)
	///signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)
	lis, err := net.Listen("tcp", GrpcAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// RabbitMQ connection
	rabbitmq, err := messaging.NewRabbitMQ(rabbitMqURI)
	if err != nil {
		log.Fatal(err)
	}
	defer rabbitmq.Close()

	log.Println("Starting RabbitMQ connection")
	///select {
	///case err := <-serverErrors:
	///	log.Printf("Error starting server: %v", err)
	publisher := events.NewTripEventPublisher(rabbitmq)
	// Start driver consumer
	driverConsumer := events.NewDriverConsumer(rabbitmq, svc)
	go driverConsumer.Listen()

	// Start payment consumer (webhook)
	paymentConsumer := events.NewPaymentConsumer(rabbitmq, svc)
	go paymentConsumer.Listen()

	// старт грпс
	grpcServer := grpcserver.NewServer(tracing.WithTracingInterceptors()...)
	grpc.NewGRPCHandler(grpcServer, svc, publisher) // прописываем хэндлер
	///case sig := <-shutdown:
	///log.Printf("Server is shutting down due to %v signal", sig)

	///ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	///defer cancel()
	log.Printf("Starting gRPC server Trip service on port %s", lis.Addr().String())
	///	if err := server.Shutdown(ctx); err != nil {
	///	log.Printf("Could not stop server gracefully: %v", err)
	///server.Close()
	///}
	///}
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Printf("failed to serve: %v", err)
			cancel()
		}
	}()

	// wait for the shutdown signal
	<-ctx.Done()
	log.Println("Shutting down the server...")
	grpcServer.GracefulStop()
}
