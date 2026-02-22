package http

import (
	"encoding/json"
	"log"
	"net/http"
	"ride-sharing/services/trip-service/internal/domain"
	"ride-sharing/shared/types"
)

// структура, отвечающая за бизнес-логику сервиса поездок
type HttpHandler struct {
	Service domain.TripService
}

type previewTripRequest struct {
	UserID      string           `json:"userID"`      //айди пользователя
	Pickup      types.Coordinate `json:"pickup"`      //точка посадки
	Destination types.Coordinate `json:"destination"` //точка назначения
}

// предварительный просмотр поездки
func (s *HttpHandler) HandleTripPreview(w http.ResponseWriter, r *http.Request) {
	var reqBody previewTripRequest
	// записываем из тела запроса напрямую в структуру
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "failed to parse JSON data", http.StatusBadRequest)
		return
	}
	// создание модели стоиимости поездки
	//fare := &domain.RideFareModel{
	//	UserID: "42", // заглушка reqBody.UserID
	//}
	// получаем контекст (позв передавать таймауты, отмену)
	ctx := r.Context()
	//формируем поездку передавая контекст и данные тарифа
	//t, err := s.Service.CreateTrip(ctx, fare)
	// передаем указатели на точку где чел и куда едет
	t, err := s.Service.GetRoute(ctx, &reqBody.Pickup, &reqBody.Destination)
	if err != nil {
		log.Println(err)
	}
	// Отправка ответа клиенту
	writeJSON(w, http.StatusOK, t)
}

// сереализуем данные в джсон
func writeJSON(w http.ResponseWriter, status int, data any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(data)
}
