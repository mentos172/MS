// бизнес логика!
package service
//сервисный слой (service), который обеспечивает работу с бизнес-логикой,
// используя репозиторий для хранения и получения данных, и взаимодействует с внешними API (OSRM).
import (
	"context"// для отмены\тайм аутов операций
	"encoding/json"//парсинг джсон
	"fmt"
	"io"//чтение данных из потока
	"net/http"// выполнение хттп запросов
	"ride-sharing/services/trip-service/internal/domain"
	"ride-sharing/shared/types"

	"go.mongodb.org/mongo-driver/bson/primitive"// генерация Object id
)
// структура содержащая репозиторий для работы с б.д
type service struct {
	repo domain.TripRepository
}
// создаем новый экземпляр сервиса, принимая любой объект удовлетворяющий интерфейсу трипрепозитори
func NewService(repo domain.TripRepository) *service {
	return &service{
		repo: repo,
	}
}
//Создает новую модель поездки (TripModel):
// ID — генерирует новый уникальный ObjectID через primitive.NewObjectID().
//UserID — берет из переданного RideFareModel.
//Status — имеет строку "pending", что означает, что поездка еще не стартовала и ожидает обработки.
//RideFare — записывает переданный объект стоимости.
//Дальше вызывает метод репозитория CreateTrip, передавая контекст и новую модель, и возвращает результат.
func (s *service) CreateTrip(ctx context.Context, fare *domain.RideFareModel) (*domain.TripModel, error) {
	t := &domain.TripModel{
		ID:       primitive.NewObjectID(),
		UserID:   fare.UserID,
		Status:   "pending",
		RideFare: fare,
	}

	return s.repo.CreateTrip(ctx, t)
}
//Этот метод запрашивает маршрут между двумя точками (начальной и конечной) у сервиса OSRM 
// и возвращает полученный ответ.
func (s *service) GetRoute(ctx context.Context, pickup, destination *types.Coordinate) (*types.OsrmApiResponse, error) {
	url := fmt.Sprintf( // формирование запроса
		"http://router.project-osrm.org/route/v1/driving/%f,%f;%f,%f?overview=full&geometries=geojson",
		pickup.Longitude, pickup.Latitude, // гео данные откуда
		destination.Longitude, destination.Latitude, // куда
	)
// отправляем запрос по сформированному URL
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch route from OSRM API: %v", err)
	}
	defer resp.Body.Close()
// читаем ответ из http запроса
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read the response: %v", err)
	}
//Создает переменную routeResp типа types.OsrmApiResponse.
//Распарсивает JSON в эту структуру.
	var routeResp types.OsrmApiResponse
	if err := json.Unmarshal(body, &routeResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}
//Возвращает указатель на структуру
	return &routeResp, nil
}
