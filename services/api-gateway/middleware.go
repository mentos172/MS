package main
// подключение корс
import "net/http"

func enableCORS(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// разрешаем доступ со всех доменов
		w.Header().Set("Access-Control-Allow-Origin", "*")
		// разрешенные методы
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		// разрешенные заголовки
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// allow preflight requests from the browser API
		// обработка предварительных запросов
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
// продолжаем выполнение обработчика
		handler(w, r)
	}
}