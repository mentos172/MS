package types

import pb "ride-sharing/shared/proto/trip"
// джсон ответ от осрм апи
type OsrmApiResponse struct {
	Routes []struct {
		Distance float64 `json:"distance"`
		Duration float64 `json:"duration"`
		Geometry struct {
			Coordinates [][]float64 `json:"coordinates"`
		} `json:"geometry"`
	} `json:"routes"`
}

func (o *OsrmApiResponse) ToProto() *pb.Route {
	route := o.Routes[0]// из массива маршрутов берем первый маршрут
	geometry := route.Geometry.Coordinates// получаем координаты маршрута
	coordinates := make([]*pb.Coordinate, len(geometry))// создаем срез указателей структуры нужной длины
	for i, coord := range geometry {// преобразование координат в протобаф
		coordinates[i] = &pb.Coordinate{ // перебераем все координаты из геометри  
			Latitude:  coord[0],
			Longitude: coord[1],
		}
	}
// возввращаем модель протобаф
	return &pb.Route{ // создаем указатель на пб роут
		Geometry: []*pb.Geometry{ //кладем координаты
			{
				Coordinates: coordinates,
			},
		},
		Distance: route.Distance,
		Duration: route.Duration,
	}
}
// структура с ценой за км и за время
type PricingConfig struct {
	PricePerUnitOfDistance float64
	PricingPerMinute       float64
}
//Возвращает указатель на новый объект PricingConfig с заданными значениями
func DefaultPricingConfig() *PricingConfig {
	return &PricingConfig{
		PricePerUnitOfDistance: 1.5,
		PricingPerMinute:       0.25,
	}
}