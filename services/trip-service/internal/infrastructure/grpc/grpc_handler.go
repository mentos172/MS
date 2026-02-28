package grpc
//реализующий gRPC-сервер для сервиса поездок (trip-service) в контексте системы ride-sharing
//Этот код реализует обработчик gRPC для сервиса поездок (TripService) с методом PreviewTrip, 
//который получает координаты начала и конца маршрута и возвращает информацию о маршруте
import (
	"context"
	"log"
	"ride-sharing/services/trip-service/internal/domain"
	pb "ride-sharing/shared/proto/trip"
	"ride-sharing/shared/types"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)
//Встраивается pb.UnimplementedTripServiceServer, что один из шаблонов реализации gRPC серверов — для удобства и соблюдения контрактов интерфейса.
//Содержит поле service типа domain.TripService — бизнес-логику, которая реализует нужные методы (например, GetRoute).
type gRPCHandler struct {
	pb.UnimplementedTripServiceServer

	service domain.TripService
}
// создание обработчика
func NewGRPCHandler(server *grpc.Server, service domain.TripService) *gRPCHandler {
	handler := &gRPCHandler{// получаем уже созданный грпс сервер и сервис биз-логики
		service: service,
	}

	pb.RegisterTripServiceServer(server, handler)//Создаёт экземпляр обработчика, регистрирует его в gRPC сервере.
	return handler//возвращаем обработчик
}

func (h *gRPCHandler) CreateTrip(ctx context.Context, req *pb.CreateTripRequest) (*pb.CreateTripResponse, error) {
	
	//return nil, status.Errorf(codes.Unimplemented, "method CreateTrip not implemented")
//получает ID стоимости, валидирует его, а затем создает поездку. 
	fareID := req.GetRideFareID()//извлечение параметров из запроса из протобаф
	userID := req.GetUserID()
//Получает объект RideFareModel из базы по fareID.
//Проверяет, что rideFare.UserID == userID, чтобы убедиться, что стоимость принадлежит текущему пользователю.
	rideFare, err := h.service.GetAndValidateFare(ctx, fareID, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to validate the fare: %v", err)
	}
//внутри вызываемой функции происходит создание новой модели Trip и сохранение её в базе.
	trip, err := h.service.CreateTrip(ctx, rideFare)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create the trip: %v", err)
	}

	// Add a comment at the end of the function to publish an event on the Async Comms module.
//возвращает ID созданной поездки в виде строки.
	return &pb.CreateTripResponse{
		TripID: trip.ID.Hex(),
	}, nil
}

func (h *gRPCHandler) PreviewTrip(ctx context.Context, req *pb.PreviewTripRequest) (*pb.PreviewTripResponse, error) {
	pickup := req.GetStartLocation()//получаем координаты где мы
	destination := req.GetEndLocation()// координаты куда
// преобразование координат из протобаф
	pickupCoord := &types.Coordinate{
		Latitude:  pickup.Latitude,
		Longitude: pickup.Longitude,
	}
	destinationCoord := &types.Coordinate{
		Latitude:  destination.Latitude,
		Longitude: destination.Longitude,
	}
	userID := req.GetUserID() // получаем айди пользователя из запроса
// через h.service.GetRoute запрашиваем маршрут между координатами
// Вызывает метод GetRoute, чтобы получить маршрут t
	route, err := h.service.GetRoute(ctx, pickupCoord, destinationCoord, true)
	if err != nil {
		log.Println(err)
		return nil, status.Errorf(codes.Internal, "failed to get route: %v", err)
	}
	// На основании маршрута t оценивает тарифы, возвращая срез []*RideFareModel
estimatedFares := h.service.EstimatePackagesPriceWithRoute(route)
	//Передает полученные оценки (estimatedFares) и ID пользователя userID.
    // В результате получает финальные тарифы (fares).
	fares, err := h.service.GenerateTripFares(ctx, estimatedFares, userID, route)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate the ride fares: %v", err)
	}

	// возвращаем ответ конвертирую в протобаф
	// Возвращает PreviewTripResponse.
    //В Route — маршрут в protobuf-формате.
    //В RideFares — список тарифов, преобразованный в protobuf.
	return &pb.PreviewTripResponse{
		Route:     route.ToProto(),
		RideFares: domain.ToRideFaresProto(fares),
	}, nil
}