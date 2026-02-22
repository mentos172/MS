package domain

import "go.mongodb.org/mongo-driver/bson/primitive"

// описываем тариф и стоимость услуг
type RideFareModel struct {
	ID                primitive.ObjectID //унникальный айди
	UserID            string             //идентиф пользователя
	PackageSlug       string             //пакет услуги
	TotalPriceInCents float64            //цена
}

//Хранит информацию о расчёте стоимости поездки.
//Используется сервисом для создания поездки (TripModel).
//Может содержать информацию о типе выбранного тарифа (через PackageSlug), что удобно для различения тарифных пакетов в бизнес-логике.
//Связан с пользователем через UserID, чтобы знать, кому относится платёж.
