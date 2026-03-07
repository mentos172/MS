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
	"ride-sharing/shared/proto/trip"
tripTypes "ride-sharing/services/trip-service/pkg/types"
	"go.mongodb.org/mongo-driver/bson/primitive"// генерация Object id
pbd "ride-sharing/shared/proto/driver"
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
// указатель на водителя
//Дальше вызывает метод репозитория CreateTrip, передавая контекст и новую модель, и возвращает результат.
func (s *service) CreateTrip(ctx context.Context, fare *domain.RideFareModel) (*domain.TripModel, error) {
	t := &domain.TripModel{
		ID:       primitive.NewObjectID(),
		UserID:   fare.UserID,
		Status:   "pending",
		RideFare: fare,
		Driver:   &trip.TripDriver{},
	}

	return s.repo.CreateTrip(ctx, t)
}
//Этот метод запрашивает маршрут между двумя точками (начальной и конечной) у сервиса OSRM 
// и возвращает полученный ответ.
///func (s *service) GetRoute(ctx context.Context, pickup, destination *types.Coordinate) (*types.OsrmApiResponse, error) {
//func (s *service) GetRoute(ctx context.Context, pickup, destination *types.Coordinate) (*tripTypes.OsrmApiResponse, error) {	
func (s *service) GetRoute(ctx context.Context, pickup, destination *types.Coordinate, useOSRMApi bool) (*tripTypes.OsrmApiResponse, error) {
	if !useOSRMApi {
		// Return a simple mock response in case we don't want to rely on an external API
		return &tripTypes.OsrmApiResponse{
			Routes: []struct {
				Distance float64 `json:"distance"`
				Duration float64 `json:"duration"`
				Geometry struct {
					Coordinates [][]float64 `json:"coordinates"`
				} `json:"geometry"`
			}{
				{
					Distance: 5.0, // 5km
					Duration: 600, // 10 minutes
					Geometry: struct {
						Coordinates [][]float64 `json:"coordinates"`
					}{
						Coordinates: [][]float64{
							{pickup.Latitude, pickup.Longitude},
							{destination.Latitude, destination.Longitude},
						},
					},
				},
			},
		}, nil
	}
	baseURL := "http://osrm.selfmadeengineer.com"
url := fmt.Sprintf( // формирование запроса
	
		"%s/route/v1/driving/%f,%f;%f,%f?overview=full&geometries=geojson",
		baseURL,
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
	///var routeResp types.OsrmApiResponse
	var routeResp tripTypes.OsrmApiResponse
	if err := json.Unmarshal(body, &routeResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}
//Возвращает указатель на структуру
	return &routeResp, nil
}

//расчет стоимости поездки и созданием тарифов, используя карту маршрута и базовые цены.


func (s *service) EstimatePackagesPriceWithRoute(route *tripTypes.OsrmApiResponse) []*domain.RideFareModel {
	baseFares := getBaseFares()
	estimatedFares := make([]*domain.RideFareModel, len(baseFares))

	for i, f := range baseFares {
		estimatedFares[i] = estimateFareRoute(f, route)
	}

	return estimatedFares
}
//Берет базовые тарифы (getBaseFares()).
//Для каждого тарифа считает примерную стоимость с учетом маршрута (estimateFareRoute).
//Возвращает список оцененных тарифов.

func (s *service) GenerateTripFares(ctx context.Context, rideFares []*domain.RideFareModel, userID string, route *tripTypes.OsrmApiResponse) ([]*domain.RideFareModel, error) {
	fares := make([]*domain.RideFareModel, len(rideFares))

	for i, f := range rideFares {
		id := primitive.NewObjectID()

		fare := &domain.RideFareModel{
			UserID:            userID,
			ID:                id,
			TotalPriceInCents: f.TotalPriceInCents,
			PackageSlug:       f.PackageSlug,
			Route: route,
		}

		if err := s.repo.SaveRideFare(ctx, fare); err != nil {
			return nil, fmt.Errorf("failed to save trip fare: %w", err)
		}

		fares[i] = fare
	}

	return fares, nil
}
//Создает новые объекты RideFareModel для каждого тарифа.
//Присваивает уникальный ID (primitive.NewObjectID()).
//Устанавливает UserID, TotalPriceInCents, PackageSlug.
//Сохраняет их в репозиторий (s.repo.SaveRideFare).
//Возвращает список созданных тарифов.
func (s *service) GetAndValidateFare(ctx context.Context, fareID, userID string) (*domain.RideFareModel, error) {
	fare, err := s.repo.GetRideFareByID(ctx, fareID)
	if err != nil {
		return nil, fmt.Errorf("failed to get trip fare: %w", err)
	}

	if fare == nil {
		return nil, fmt.Errorf("fare does not exist")
	}

	// User fare validation (user is owner of this fare?)
	if userID != fare.UserID {
		return nil, fmt.Errorf("fare does not belong to the user")
	}

	return fare, nil
}//Вызывает репозиторий для получения стоимости по fareID.
//Проверяет наличие ошибки при вызове.
//Проверяет, что fare действительно получен (если nil — ошибка).
//Проверяет, что userID совпадает с fare.UserID, чтобы убедиться, что пользователь владеет этой стоимостью.
//Возвращает объект fare, если все проверки прошли успешно.

func estimateFareRoute(f *domain.RideFareModel, route *tripTypes.OsrmApiResponse) *domain.RideFareModel {
	pricingCfg := tripTypes.DefaultPricingConfig()
	carPackagePrice := f.TotalPriceInCents

	distanceKm := route.Routes[0].Distance
	durationInMinutes := route.Routes[0].Duration

	distanceFare := distanceKm * pricingCfg.PricePerUnitOfDistance
	timeFare := durationInMinutes * pricingCfg.PricingPerMinute
	totalPrice := carPackagePrice + distanceFare + timeFare

	return &domain.RideFareModel{
		TotalPriceInCents: totalPrice,
		PackageSlug:       f.PackageSlug,
	}
}
//Получает конфигурацию цен (DefaultPricingConfig()), которая содержит цену за км и минуту.
//Из маршрута извлекает расстояние и длительность.
//Расчет стоимости:
//distanceFare — цена за расстояние.
//timeFare — цена за время.
//Итоговая — сумма базовой цены (f.TotalPriceInCents), расстояния и времени.
//Возвращает новый объект RideFareModel с рассчитанной ценой и тем же пакетом.
func getBaseFares() []*domain.RideFareModel {
	return []*domain.RideFareModel{
		{
			PackageSlug:       "suv",
			TotalPriceInCents: 200,
		},
		{
			PackageSlug:       "sedan",
			TotalPriceInCents: 350,
		},
		{
			PackageSlug:       "van",
			TotalPriceInCents: 400,
		},
		{
			PackageSlug:       "luxury",
			TotalPriceInCents: 1000,
		},
	}
}
func (s *service) GetTripByID(ctx context.Context, id string) (*domain.TripModel, error) {
	return s.repo.GetTripByID(ctx, id)
}

func (s *service) UpdateTrip(ctx context.Context, tripID string, status string, driver *pbd.Driver) error {
	return s.repo.UpdateTrip(ctx, tripID, status, driver)
}
//Возвращает набор базовых предложений с фиксированными ценами.


//EstimatePackagesPriceWithRoute — оценивает тарифы на основе маршрута.
//GenerateTripFares — создает финальные тарифы для каждого пакета и сохраняет их в репозиторий.
//estimateFareRoute — рассчитывает цену для отдельного тарифного пакета, учитывая маршрут.
//getBaseFares — возвращает базовые цены для разных пакетов.