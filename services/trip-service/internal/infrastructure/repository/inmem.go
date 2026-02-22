package repository
//реализация интерфейса TripRepository (или его аналог), использующая внутренние карты (map) в памяти вместо базы данных.
//  Полезно для быстрого прототипирования или тестирования.
import (
	"context"
	"ride-sharing/services/trip-service/internal/domain" //указали путь к файлу бизнес модели
)
// структура реализующая интерфейс репозитория (для работы без базы данных)
type inmemRepository struct {
	trips     map[string]*domain.TripModel //хранение поездки, ключ строка, значение на указатель трип модел
	rideFares map[string]*domain.RideFareModel//храниние стоимости
}
// Создает новый объект типа *inmemRepository (указатель на структуру).
//Внутри этого объекта:
//Поле trips — инициализируется новой пустой картой map[string]*domain.TripModel.
//Поле rideFares — тоже инициализируется пустой картой.
func NewInmemRepository() *inmemRepository {
	return &inmemRepository{
		trips:     make(map[string]*domain.TripModel),
		rideFares: make(map[string]*domain.RideFareModel),
	}
}
// добавляем новую поездку в карту трип, 
func (r *inmemRepository) CreateTrip(ctx context.Context, trip *domain.TripModel) (*domain.TripModel, error) {
	r.trips[trip.ID.Hex()] = trip //  преобразует ObjectID в строку вида "507f1f77bcf86cd799439011"
	return trip, nil
}
