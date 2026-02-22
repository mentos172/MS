package types

// маршрут с длинной временем массивовом указателей на обьекты
type Route struct {
	Distance float64     `json:"distance"`
	Duration float64     `json:"duration"`
	Geometry []*Geometry `json:"geometry"`
}

// описываем часть маршрута как набор точек координат
type Geometry struct {
	Coordinates []*Coordinate `json:"coordinates"`
}

// описание гео точки широтой и долготой
type Coordinate struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// структура для прямого отображения джсон ответа
// от osrm служба построения маршрута
type OsrmApiResponse struct {
	Routes []struct {
		Distance float64 `json:"distance"`
		Duration float64 `json:"duration"`
		Geometry struct {
			Coordinates [][]float64 `json:"coordinates"`
		} `json:"geometry"`
	} `json:"routes"`
}
