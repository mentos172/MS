package main

import (
	"context"
	"log"      //создание логов
	"net/http" // работа с созданием сервера
	"os"
	"os/signal"
	"syscall"
	"time"
"ride-sharing/shared/messaging"
	"ride-sharing/shared/env" //модуль в котором переменные
)

var (
	httpAddr = env.GetString("HTTP_ADDR", ":8081") // читаем переменну, и если ее нет то случаем порт
rabbitMqURI = env.GetString("RABBITMQ_URI", "amqp://guest:guest@rabbitmq:5672/")
//Вы получаете значение переменной окружения "RABBITMQ_URI".
//Если переменная не установлена, то используется значение по умолчанию:
//"amqp://guest:guest@rabbitmq:5672/".
)

func main() {
	log.Println("Starting API Gateway")
	//запускаем сервер и на все запросы отвечаем хэло
	mux := http.NewServeMux()  
	// RabbitMQ connection
	rabbitmq, err := messaging.NewRabbitMQ(rabbitMqURI)
	if err != nil {
		log.Fatal(err)
	}
	defer rabbitmq.Close()

	log.Println("Starting RabbitMQ connection")                              //вместо хттп встроенный мультиплексер
	mux.HandleFunc("POST /trip/preview", enableCORS(handleTripPreview))// подключаем корс//mux.HandleFunc(" POST /trip/preview", handleTripPreview) //  func(w http.ResponseWriter, r *http.Request) убрали это с кода и создали файл в это директории http
	mux.HandleFunc("POST /trip/start", enableCORS(handleTripStart)) // начало путеш
	// веб сокет для водителя и для челикса
	mux.HandleFunc("/ws/drivers", func(w http.ResponseWriter, r *http.Request) {
		handleDriversWebSocket(w, r, rabbitmq)
	})
	mux.HandleFunc("/ws/riders", func(w http.ResponseWriter, r *http.Request) {
		handleRidersWebSocket(w, r, rabbitmq)
	})
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
