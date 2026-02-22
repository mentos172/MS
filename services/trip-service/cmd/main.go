// запускает HTTP сервер для сервиса,
// связанного с поездками (trip-service) в системе
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	h "ride-sharing/services/trip-service/internal/infrastructure/http"     // обработчик
	"ride-sharing/services/trip-service/internal/infrastructure/repository" //слой доступа к данным
	"ride-sharing/services/trip-service/internal/service"
	"syscall"
	"time"
)

func main() {
	//ctx := context.Background() во 2 версии убирается

	inmemRepo := repository.NewInmemRepository() //создание репозитория который хранит данные

	svc := service.NewService(inmemRepo) // создаем экземпляр сервиса биз логики - он принемает репозиторий
	mux := http.NewServeMux()
	//fare := &domain.RideFareModel{
	//	UserID: "42",   использовали в проверке
	//}
	httphandler := h.HttpHandler{Service: svc} //инициализация  обработчика

	//t, err := svc.CreateTrip(ctx, fare) //вызов метода создания поездки
	//if err != nil {
	//	log.Println(err)
	//}
	mux.HandleFunc("POST /preview", httphandler.HandleTripPreview)
	//log.Println(t) //выводим созданную поездку
	server := &http.Server{
		Addr:    ":8083",
		Handler: mux,
	}
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
	serverErrors := make(chan error, 1)

	go func() {
		log.Printf("Server listening on %s", server.Addr)
		serverErrors <- server.ListenAndServe()
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		log.Printf("Error starting server: %v", err)

	case sig := <-shutdown:
		log.Printf("Server is shutting down due to %v signal", sig)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Could not stop server gracefully: %v", err)
			server.Close()
		}
	}
}
