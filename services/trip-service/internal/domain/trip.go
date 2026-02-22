package domain
// часть домена (часть бизнес-логики приложения), 
// связанный с моделями поездки
import (
	"context"
	"ride-sharing/shared/types"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TripModel struct { //описание поездки 
	ID       primitive.ObjectID // уникальный айди (авто)
	UserID   string //айди пользователя
	Status   string // статус поездки ( креэйтинг, ин прогресс ..)
	RideFare *RideFareModel // указатель на модель с информацией о стоимости поездки (RideFareModel),
}
//интерфейс для определения работы с базой данных
type TripRepository interface {
	//создает новую поездку и 
	CreateTrip(ctx context.Context, trip *TripModel) (*TripModel, error)// возвращаем модель с присвоенным айди
	// трип модел указатель на модель который надо сохранить
}
// интерфейс описывает бизнес логику как работать с поездками
type TripService interface {
	CreateTrip(ctx context.Context, fare *RideFareModel) (*TripModel, error)// формирование поездки основываясь на тариф
	GetRoute(ctx context.Context, pickup, destination *types.Coordinate) (*types.OsrmApiResponse, error)
	//построение маршрута осовываясь на координаты (между точками)
}
