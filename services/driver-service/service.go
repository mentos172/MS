package main
// у вас есть структура данных,
//  которая хранит информацию о водителях, и функции для создания этой структуры.
import (
	math "math/rand/v2"
	pb "ride-sharing/shared/proto/driver"
	"ride-sharing/shared/util"
	"sync"

	"github.com/mmcloughlin/geohash"
)
//Driver — это указатель на protobuf struct Driver, который, 
// скорее всего, содержит данные о водителе (ID, имя, статус, координаты и т.п.).
type driverInMap struct {
	Driver *pb.Driver
	// Index int
	// TODO: route
}
 // хранение водителей
 // список водителей ([]*driverInMap), то есть — массив указателей.
type Service struct {
	drivers []*driverInMap
	mu      sync.RWMutex
}
// создание нового сервиса с пустым списком водителей
func NewService() *Service {
	return &Service{
		drivers: make([]*driverInMap, 0),
	}
}
//поиск и возврат подходящего (первого попавш) водителя
func (s *Service) FindAvailableDrivers(packageType string) []string {
	var matchingDrivers []string

	for _, driver := range s.drivers {
		if driver.Driver.PackageSlug == packageType {
			matchingDrivers = append(matchingDrivers, driver.Driver.Id)
		}
	}

	if len(matchingDrivers) == 0 {
		return []string{}
	}

	return matchingDrivers
}
//Service — это структура, которая содержит список драйверов drivers (пример — список активных водителей).
//Этот список защищен мьютексом mu для безопасной работы в многопоточной среде.
//В функции регистрируется новый драйвер и добавляется в список, а при удалении — он удаляется из этого списка.

func (s *Service) RegisterDriver(driverId string, packageSlug string) (*pb.Driver, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
//Блокирует мьютекс mu для предотвращения одновременных изменений коллекции.
//defer гарантирует автомат освобождение мьютекса по завершении функции.
	randomIndex := math.IntN(len(PredefinedRoutes))
	randomRoute := PredefinedRoutes[randomIndex]// выбор случайного маршрута

	randomPlate := GenerateRandomPlate() //случайный номер авто
	randomAvatar := util.GetRandomAvatar(randomIndex)//случайный аватар

	// we can ignore this property for now, but it must be sent to the frontend.

	//Конвертирует координаты первой точки маршрута в geohash — компактное кодирование географической позиции.
	geohash := geohash.Encode(randomRoute[0][0], randomRoute[0][1])
// формирование данных водителя
	driver := &pb.Driver{
		Id:             driverId,
		Geohash:        geohash,
		Location:       &pb.Location{Latitude: randomRoute[0][0], Longitude: randomRoute[0][1]},
		Name:           "Lando Norris",
		PackageSlug:    packageSlug,
		ProfilePicture: randomAvatar,
		CarPlate:       randomPlate,
	}

//добавляем в слайс
	s.drivers = append(s.drivers, &driverInMap{
		Driver: driver,
	})
// возвращаем результат
	return driver, nil
}
//Блокировка mu для безопасности.
//Проходит по списку s.drivers.
//Ищет совпадение по driverId.
//В случае совпадения — удаляет элемент из слайса
func (s *Service) UnregisterDriver(driverId string) {
	s.mu.Lock()
	defer s.mu.Unlock()


	//Удаление элемента выполнено через срез, который исключает текущий элемент
	for i, driver := range s.drivers {
		if driver.Driver.Id == driverId {
			s.drivers = append(s.drivers[:i], s.drivers[i+1:]...)
		}
	}
}