package grpc_clients

import (
	"os"
	pb "ride-sharing/shared/proto/trip"

	"google.golang.org/grpc"
		"google.golang.org/grpc/credentials/insecure"
)

type tripServiceClient struct {
	Client pb.TripServiceClient // обьект интерфейса grpc
	conn   *grpc.ClientConn // conn обьект соединения *grpc... канал соедиения
}

func NewTripServiceClient() (*tripServiceClient, error) {
	tripServiceURL := os.Getenv("TRIP_SERVICE_URL")// проверяем переменную чтоб получить адрес сервера
	if tripServiceURL == "" {
		tripServiceURL = "trip-service:9093"// если переменная пустая то используем 
	}

	//conn, err := grpc.NewClient(tripServiceURL)// вызов который установливает grpc соединение
	conn, err := grpc.NewClient(tripServiceURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	client := pb.NewTripServiceClient(conn)//Использует пакет pb 
	// (обычно автоматически сгенерированный из protobuf), чтобы создать клиент для TripService.
// Передает в конструктор соединение conn
// Создает новую структуру tripServiceClient, передавая туда созданный gRPC-клиент и соединение.
	return &tripServiceClient{
		Client: client,
		conn:   conn,
	}, nil// возвращаем нил типа успешно
}
//закрываем соединение и если есть ошибка игнорим ее
func (c *tripServiceClient) Close() {
	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			return
		}
	}
}