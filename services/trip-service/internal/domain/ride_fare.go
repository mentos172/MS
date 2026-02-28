package domain

import (
	"ride-sharing/services/trip-service/pkg/types"
	"go.mongodb.org/mongo-driver/bson/primitive"
pb "ride-sharing/shared/proto/trip")
// описываем тариф и стоимость услуг
type RideFareModel struct {
	ID                primitive.ObjectID //унникальный айди
	UserID            string             //идентиф пользователя
	PackageSlug       string             //пакет услуги
	TotalPriceInCents float64  //цена
	Route             *types.OsrmApiResponse          
}

//Хранит информацию о расчёте стоимости поездки.
//Используется сервисом для создания поездки (TripModel).
//Может содержать информацию о типе выбранного тарифа (через PackageSlug), что удобно для различения тарифных пакетов в бизнес-логике.
//Связан с пользователем через UserID, чтобы знать, кому относится платёж.


// преобразования внутренних моделей в протобуферные структуры для обмена данными.
func (r *RideFareModel) ToProto() *pb.RideFare {
	return &pb.RideFare{
		Id:                r.ID.Hex(),
		UserID:            r.UserID,
		PackageSlug:       r.PackageSlug,
		TotalPriceInCents: r.TotalPriceInCents,
		
	}
}
// преобразование внутренней мождели для протобаф
func ToRideFaresProto(fares []*RideFareModel) []*pb.RideFare {
	var protoFares []*pb.RideFare
	for _, f := range fares {
		protoFares = append(protoFares, f.ToProto())
	}
	return protoFares
}
// Эта функция принимает список (срез) моделей RideFareModel.
//Итеративно вызывает у каждого элемента метод ToProto().
//Собирает результат в новый срез []*pb.RideFare.
//Возвращает его — это удобно для отправки данных через API или gRPC.