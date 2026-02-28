package domain
// часть домена (часть бизнес-логики приложения), 
// связанный с моделями поездки
import (
	"context"
	"ride-sharing/shared/types"
tripTypes "ride-sharing/services/trip-service/pkg/types"
pb "ride-sharing/shared/proto/trip"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TripModel struct { //описание поездки 
	ID       primitive.ObjectID // уникальный айди (авто)
	UserID   string //айди пользователя
	Status   string // статус поездки ( креэйтинг, ин прогресс ..)
	RideFare *RideFareModel // указатель на модель с информацией о стоимости поездки (RideFareModel),
Driver   *pb.TripDriver// указатель на водителя используя протобаф
}
//интерфейс для определения работы с базой данных
type TripRepository interface {
	//создает новую поездку и 
	CreateTrip(ctx context.Context, trip *TripModel) (*TripModel, error)// возвращаем модель с присвоенным айди
	// трип модел указатель на модель который надо сохранить
	SaveRideFare(ctx context.Context, f *RideFareModel) error
	GetRideFareByID(ctx context.Context, id string) (*RideFareModel, error)// возврат стоимостипо айди
}
// интерфейс описывает бизнес логику как работать с поездками
type TripService interface {
	CreateTrip(ctx context.Context, fare *RideFareModel) (*TripModel, error)// формирование поездки основываясь на тариф
	///GetRoute(ctx context.Context, pickup, destination *types.Coordinate) (*types.OsrmApiResponse, error)
	//построение маршрута осовываясь на координаты (между точками)
GetRoute(ctx context.Context, pickup, destination *types.Coordinate, useOsrmApi bool) (*tripTypes.OsrmApiResponse, error)
EstimatePackagesPriceWithRoute(route *tripTypes.OsrmApiResponse) []*RideFareModel
// Функция принимает route — скорее всего, ответ от API маршрутизации OSRM 
// Возвращает список объектов RideFareModel — модели, содержащие информацию о стоимости по определенному маршруту.
	GenerateTripFares(ctx context.Context, fares []*RideFareModel, userID string,Route *tripTypes.OsrmApiResponse) ([]*RideFareModel, error)
// Берет уже подготовленный список fares и идентификатор пользователя userID.
// Генерирует "финальные" тарифы для поездки — возможно, добавляет дополнительные параметры, сохраняет в базу и т.п.
// Возвращает обновленный список RideFareModel или ошибку.
GetAndValidateFare(ctx context.Context, fareID, userID string) (*RideFareModel, error)
//получить информацию о стоимости (RideFareModel) по fareID, 
//проверить её валидность, и возможно убедиться, что она принадлежит указанному пользователю (userID).
}

