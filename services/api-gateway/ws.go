package main

import (
	"encoding/json"
	"log"
	"net/http"
	"ride-sharing/services/api-gateway/grpc_clients"
	"ride-sharing/shared/contracts"
	"ride-sharing/shared/messaging"
	"ride-sharing/shared/proto/driver"

	//"github.com/gorilla/websocket"
)

/// упгрейдер, объект который переводит хттп на вебсокет
/// после добавления вебсокет конечкшона не нужен
///var upgrader = websocket.Upgrader{
	///CheckOrigin: func(r *http.Request) bool {
		///return true
	///},

	var (
	connManager = messaging.NewConnectionManager()
)
// чекориджин установлен в тру значит что любые источники могут подключаться
// функция которая обрабатывает соединение вызывается при подключении пасс
func handleRidersWebSocket(w http.ResponseWriter, r *http.Request, rb *messaging.RabbitMQ) {
	//упгрейдин с хттп до веб сокета если ошибка то логируем
	conn, err := connManager.Upgrade(w, r)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	// после выхода из функции соед закрывается
	defer conn.Close()
	//подключаем юзер айди из урл
	userID := r.URL.Query().Get("userID")
	if userID == "" {
		log.Println("No user ID provided")
		return
	}
		// Add connection to manager
	connManager.Add(userID, conn)
	defer connManager.Remove(userID)
		// Initialize queue consumers
		//  создаетс очередь с ключем NotifyDriverNoDriversFoundQueue.
	queues := []string{
		messaging.NotifyDriverNoDriversFoundQueue,
		messaging.NotifyDriverAssignQueue, // ключ для водителя
	}
//для каждой очереди создаете консьюмера (потребителя) с помощью функции NewQueueConsumer,
//  затем запускаете его методом Start(). — Если запуск неудачен — выводите ошибку в лог.
	for _, q := range queues {
		consumer := messaging.NewQueueConsumer(rb, connManager, q)

		if err := consumer.Start(); err != nil {
			log.Printf("Failed to start consumer for queue: %s: err: %v", q, err)
		}
	}
	// бесконечный цикл
	// читаем сообщения от клиента
	// при ошибке цикл завершится
	// логируем сообщения
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading message: %v", err)
			break
		}

		log.Printf("Received message: %s", message)
	}
}

// обработчик для водителя
// http.ResponseWriter это хттп интерфейсы
func handleDriversWebSocket(w http.ResponseWriter, r *http.Request, rb *messaging.RabbitMQ) {
	conn, err := connManager.Upgrade(w, r)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	defer conn.Close()
	// получаем айди юзера
	userID := r.URL.Query().Get("userID")
	if userID == "" {
		log.Println("No user ID provided")
		return
	}
	//тариф
	packageSlug := r.URL.Query().Get("packageSlug")
	if packageSlug == "" {
		log.Println("No package slug provided")
		return
	}
	// Add connection to manager
	connManager.Add(userID, conn)
	//структура описывающая данные водителя
	///type Driver struct {
		///Id             string `json:"id"`
		///Name           string `json:"name"`
		///ProfilePicture string `json:"profilePicture"`
		///CarPlate       string `json:"carPlate"`
		///PackageSlug    string `json:"packageSlug"`
	ctx := r.Context()
//вызывается функция для создания клиента, которая подключается к сервису драйверов.
	driverService, err := grpc_clients.NewDriverServiceClient()
	if err != nil {
		log.Fatal(err)
	}

	// Closing connections
	defer func() {
		connManager.Remove(userID)
		driverService.Client.UnregisterDriver(ctx, &driver.RegisterDriverRequest{
			DriverID:    userID,
			PackageSlug: packageSlug,
		})

		driverService.Close()
		
		log.Println("Driver unregistered: ", userID)
	}()
//регистрация водителя
	driverData, err := driverService.Client.RegisterDriver(ctx, &driver.RegisterDriverRequest{
		DriverID:    userID,
		PackageSlug: packageSlug,
	})
	if err != nil {
		log.Printf("Error registering driver: %v", err)
		return
	}
	//сообщение для клиента
	//msg := contracts.WSMessage{
	//	Type: "driver.cmd.register",
		if err := connManager.SendMessage(userID, contracts.WSMessage{
		Type: contracts.DriverCmdRegister,
		
		///Data: Driver{
			///Id:             userID,
			///Name:           "Tiago",
			///ProfilePicture: util.GetRandomAvatar(1),
			///CarPlate:       "ABC123",
		///	PackageSlug:    packageSlug,
		///},
		Data: driverData.Driver,
	
	// отправляем сообщение клиенту
	//if err := conn.WriteJSON(msg); err != nil {
	}); err != nil {
		log.Printf("Error sending message: %v", err)
		return
	}
	// Initialize queue consumers
	//Создаёт список очередей (queues) — в ваш список входит messaging.DriverCmdTripRequestQueue.
//Перебирает все очереди в цикле for.
//Для каждой очереди:
//Создаёт потребителя (consumer) через messaging.NewQueueConsumer, передавая:
//rb — видимо, это ваш RabbitMQ канал или соединение,
//connManager — менеджер соединений,
//q — название очереди.
//После этого запускает потребителя командой consumer.Start().
//Проверяет ошибку и, если есть, логирует сообщение о неудаче запуска.
	queues := []string{
		messaging.DriverCmdTripRequestQueue,
	}

	for _, q := range queues {
		consumer := messaging.NewQueueConsumer(rb, connManager, q)

		if err := consumer.Start(); err != nil {
			log.Printf("Failed to start consumer for queue: %s: err: %v", q, err)
		}
	}
	// цикл для получения сообщений от водителя
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading message: %v", err)
			break
		}
//обрабатывает входящие сообщения из очереди, разбирает их и,
//  в зависимости от типа, либо игнорирует, либо пересылает дальше.
//Описывает сообщение, которое ожидается в формате JSON.
//Type — строка, указывающая тип сообщения.
//Data — сырые данные сообщения, представленные как JSON (json.RawMessage), 
//чтобы можно было их позже обработать или переслать без предварительного разбора.
		type driverMessage struct {
			Type string          `json:"type"`
			Data json.RawMessage `json:"data"`
		}
//Полученное сообщение (message) — байтовое поле.
//Пытаемся распарсить его в структуру driverMsg.
//В случае ошибки — логируем и переходим к следующему сообщению.

		var driverMsg driverMessage
		if err := json.Unmarshal(message, &driverMsg); err != nil {
			log.Printf("Error unmarshaling driver message: %v", err)
			continue
		}

		// Handle the different message type
		//V
		switch driverMsg.Type {
		case contracts.DriverCmdLocation:
			// Handle driver location update in the future
			// местоположение водителя не юзаем сча
			continue
		case contracts.DriverCmdTripAccept, contracts.DriverCmdTripDecline:
			// Forward the message to RabbitMQ
			//Эти типы сообщений пересылаются в систему RabbitMQ для дальнейшей обработки.
//Используют функцию rb.PublishMessage, передавая:
//ctx — контекст выполнения.
//driverMsg.Type — тип сообщения.
//contracts.AmqpMessage — структура сообщения для публикации, включающая:
//OwnerID — идентификатор владельца (например, пользователя или водителя).
//Data — сырые данные сообщения.
			if err := rb.PublishMessage(ctx, driverMsg.Type, contracts.AmqpMessage{
				OwnerID: userID,
				Data:    driverMsg.Data,
			}); err != nil {
				log.Printf("Error publishing message to RabbitMQ: %v", err)
			}
		default:
			log.Printf("Unknown message type: %s", driverMsg.Type)
		}
	}
}
