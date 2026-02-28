package grpc_clients
//os — для чтения переменных окружения.
//pb — это сгенерированный код protobuf, содержащий интерфейсы и сообщения для сервиса драйверов.
//grpc — основной пакет gRPC для Go.
//credentials/insecure — для установки соединения без шифрования 
//(используется в тестах или внутри безопасных сетей).
import (
	"os"
	pb "ride-sharing/shared/proto/driver"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)
//Client — это клиентский интерфейс, сгенерированный protobuf, который содержит методы вызова RPC.
//conn — сама gRPC-соединение.
type driverServiceClient struct {
	Client pb.DriverServiceClient
	conn   *grpc.ClientConn
}
//Получает URL сервиса драйверов из переменной окружения DRIVER_SERVICE_URL.
//Если переменная не задана — использует дефолтный адрес driver-service:9092.
//Создает gRPC-соединение с этим адресом conn, указывая insecure — то есть без шифрования.
//Далее создает клиентский интерфейс pb.NewDriverServiceClient(conn).
//Этот интерфейс содержит методы для вызова RPC-методов (например, RegisterDriver, UnregisterDriver и др.).
//Возвращает экземпляр driverServiceClient с готовым подключением и клиентом.
func NewDriverServiceClient() (*driverServiceClient, error) {
	driverServiceURL := os.Getenv("DRIVER_SERVICE_URL")
	if driverServiceURL == "" {
		driverServiceURL = "driver-service:9092"
	}

	conn, err := grpc.NewClient(driverServiceURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	client := pb.NewDriverServiceClient(conn)

	return &driverServiceClient{
		Client: client,
		conn:   conn,
	}, nil
}
// закрываем грпс соеденение
func (c *driverServiceClient) Close() {
	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			return
		}
	}
}