package main

import (
	///"bytes" // поток - передача данных по частям
	"encoding/json"
	"log"
	"net/http"
	"ride-sharing/shared/contracts"
	"ride-sharing/services/api-gateway/grpc_clients"
)

func handleTripStart(w http.ResponseWriter, r *http.Request) {
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

	trip, err := tripService.Client.CreateTrip(r.Context(), reqBody.toProto())
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
	tripPreview, err := tripService.Client.PreviewTrip(r.Context(), reqBody.toProto())// работа через грпс
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
