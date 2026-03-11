package main

import (
	///"bytes" // поток - передача данных по частям
	"encoding/json"
	"io"
	"log"
	"net/http"
	"ride-sharing/services/api-gateway/grpc_clients"
	"ride-sharing/shared/contracts"
	"ride-sharing/shared/env"
	"ride-sharing/shared/messaging"

	"ride-sharing/shared/tracing"

	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/webhook"
)

var tracer = tracing.GetTracer("api-gateway") //tracing

func handleTripStart(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "handleTripStart")
	defer span.End()
	var reqBody startTripRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "failed to parse JSON data", http.StatusBadRequest)
		return
	}

	defer r.Body.Close()

	// Why we need to create a new client for each connection:
	// because if a service is down, we don't want to block the whole application
	// so we create a new client for each connection
	tripService, err := grpc_clients.NewTripServiceClient()
	if err != nil {
		log.Fatal(err)
	}

	// Don't forget to close the client to avoid resource leaks!
	defer tripService.Close()

	trip, err := tripService.Client.CreateTrip(ctx, reqBody.toProto())
	if err != nil {
		log.Printf("Failed to start a trip: %v", err)
		http.Error(w, "Failed to start trip", http.StatusInternalServerError)
		return
	}

	response := contracts.APIResponse{Data: trip}

	writeJSON(w, http.StatusCreated, response)
}

// обработчик получает данные для ответа клиенту и полученный запрос
func handleTripPreview(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "handleTripPreview") //traving
	defer span.End()
	var reqBody previewTripRequest //считываем тело запроса и распарсиваем
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "failed to parse JSON data", http.StatusBadRequest)
		return
	}

	defer r.Body.Close()

	// validation проверка что обязательно есть поле юзер айди
	if reqBody.UserID == "" {
		http.Error(w, "user ID is required", http.StatusBadRequest)
		return
	}
	// преобразуем рег боди обратно в джссон
	///jsonBody, _ := json.Marshal(reqBody)
	///reader := bytes.NewReader(jsonBody) // создаем поток для чтени этого джсон
	// Why we need to create a new client for each connection:
	// because if a service is down, we don't want to block the whole application
	// so we create a new client for each connection
	tripService, err := grpc_clients.NewTripServiceClient()
	if err != nil {
		log.Fatal(err)
	}

	// Don't forget to close the client to avoid resource leaks!
	defer tripService.Close()

	// TODO: Call trip service
	// отправляем запрос к другому сервису передавая уже джсон тело с инфой из изначального запроса
	///resp, err := http.Post("http://trip-service:8083/preview", "application/json", reader)
	tripPreview, err := tripService.Client.PreviewTrip(ctx, reqBody.toProto()) // работа через грпс
	if err != nil {
		///log.Print(err)
		///return
		///}

		///defer resp.Body.Close()
		//деколируем json ответ который пришел от трип сервиса
		// если парсинга не удался то ошибка

		///var respBody any
		///if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		///http.Error(w, "failed to parse JSON data from trip service", http.StatusBadRequest)
		log.Printf("Failed to preview a trip: %v", err)
		http.Error(w, "Failed to preview trip", http.StatusInternalServerError)
		return
	}

	//response := contracts.APIResponse{Data: "ok"}
	//отвнт апи, передает обьект
	///response := contracts.APIResponse{Data: respBody}
	response := contracts.APIResponse{Data: tripPreview}
	// Отправляем клиенту HTTP ответ с кодом 201 Created,
	// тело — JSON с response.
	// сереализация
	writeJSON(w, http.StatusCreated, response)
}

//То есть, парсинг — это чтение и разбор входных данных.
//Преобразование JSON-текста в структуру Go
//  (через json.Unmarshal или json.Decoder) — это парсинг.
//парсинг преобразование текста в обьект

//.Парсинг (Parsing)	Преобразование текста в объект
// Декодировать JSON в структуру Go
//.Распарсить	То же, в разговорной речи
// "Распарсить JSON"
//.Сериализация (Marshal)	Преобразование объекта в текст
// Записать структуру в JSON
//.Десериализация (Unmarshal)	Преобразование текста в объект
// 	Считать JSON в структуру Go

func handleStripeWebhook(w http.ResponseWriter, r *http.Request, rb *messaging.RabbitMQ) {
	ctx, span := tracer.Start(r.Context(), "handleTripPreview")
	defer span.End()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	webhookKey := env.GetString("STRIPE_WEBHOOK_KEY", "")
	if webhookKey == "" {
		log.Printf("Webhook key is required")
		return
	}

	event, err := webhook.ConstructEventWithOptions(
		body,
		r.Header.Get("Stripe-Signature"),
		webhookKey,
		webhook.ConstructEventOptions{
			IgnoreAPIVersionMismatch: true,
		},
	)
	if err != nil {
		log.Printf("Error verifying webhook signature: %v", err)
		http.Error(w, "Invalid signature", http.StatusBadRequest)
		return
	}

	log.Printf("Received Stripe event: %v", event)

	switch event.Type {
	case "checkout.session.completed":
		var session stripe.CheckoutSession

		err := json.Unmarshal(event.Data.Raw, &session)
		if err != nil {
			log.Printf("Error parsing webhook JSON: %v", err)
			http.Error(w, "Invalid payload", http.StatusBadRequest)
			return
		}

		payload := messaging.PaymentStatusUpdateData{
			TripID:   session.Metadata["trip_id"],
			UserID:   session.Metadata["user_id"],
			DriverID: session.Metadata["driver_id"],
		}

		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			log.Printf("Error marshalling payload: %v", err)
			http.Error(w, "Failed to marshal payload", http.StatusInternalServerError)
			return
		}

		message := contracts.AmqpMessage{
			OwnerID: session.Metadata["user_id"],
			Data:    payloadBytes,
		}

		if err := rb.PublishMessage(
			ctx,
			contracts.PaymentEventSuccess,
			message,
		); err != nil {
			log.Printf("Error publishing payment event: %v", err)
			http.Error(w, "Failed to publish payment event", http.StatusInternalServerError)
			return
		}
	}
}
