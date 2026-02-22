package main

import (
	"encoding/json"
	"net/http"
)
// интерфейс для ответа клиенту в хттп сервере, статус хранит номер ошибки, дата любую инфу
func writeJSON(w http.ResponseWriter, status int, data any) error {
	w.Header().Set("Content-Type", "application/json")// устанавливает хттп заголовок
	w.WriteHeader(status)// записывает статус сервера
	return json.NewEncoder(w).Encode(data)// создаем новый джсон с ответом W. encode data превращает нашу
	// инфу дата в джсон и записывает в W, возвращает ошибку
}
