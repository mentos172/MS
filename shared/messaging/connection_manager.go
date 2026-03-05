package messaging
// Этот пакет реализует управление WebSocket
//  соединениями для системы обмена сообщениями 
import (
	"errors"
	"log"
	"net/http"
	"sync"

	"ride-sharing/shared/contracts"

	"github.com/gorilla/websocket"
)
// ошибка которая возвращается если соединение
// по заданному айди не найдено
var (
	ErrConnectionNotFound = errors.New("connection not found")
)

//connWrapper — это обертка над соединением WebSocket, позволяющая выполнять потокобезопасные операции.
// Это необходимо, поскольку соединение WebSocket не является потокобезопасным.
type connWrapper struct {
	conn  *websocket.Conn // указатель на вебсокет
	mutex sync.Mutex //мьютекс для синхронизации
	//Это решение — обёрнуть соединение с мьютексом — защищает от одновременных 
	//записей в соединение из разных горутин.
}
// тип управляющий соединениями
type ConnectionManager struct {
	connections map[string]*connWrapper // Local connections storage (userId -> connection)
	//хранилище соединений, где ключ — идентификатор пользователя, значение — обёрнутый websocket соединение.
	mutex       sync.RWMutex// потокобезопасность
}
//объект для апгрейда HTTP соединения до WebSocket, с функцией CheckOrigin, 
//которая разрешает все происхождения 
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for now
		//разрешает подключение со всех источников
	},
}

// Note that on multiple instances of the API gateway, the connection manager needs to store the connections on a separate shared storage.
//Создаёт и возвращает новый экземпляр менеджера соединений.
func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		connections: make(map[string]*connWrapper),
	}
}
//Принимает HTTP-запрос и возвращает апгрейдённое WebSocket соединение.
func (cm *ConnectionManager) Upgrade(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}
	return conn, nil
}
//Добавляет новое соединение:
//блокирует mutex - чтоб исключить гонки
//сохраняет соединение по id в карту
//логирует добавление.
func (cm *ConnectionManager) Add(id string, conn *websocket.Conn) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	cm.connections[id] = &connWrapper{
		conn:  conn,
		mutex: sync.Mutex{},
	}

	log.Printf("Added connection for user %s", id)
}
//Удаляет соединение по id.
//Блокирует мьютекс для потокобезопасности.
//Выполняет удаление из карты.
func (cm *ConnectionManager) Remove(id string) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	delete(cm.connections, id)
}
//возвращает websocket соединение по id: если таковое есть
//блокирует RLock для чтения
//возвращает соединение или false, если не найдено.
func (cm *ConnectionManager) Get(id string) (*websocket.Conn, bool) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	wrapper, exists := cm.connections[id]
	if !exists {
		return nil, false
	}
	return wrapper.conn, true
}
//Отправляет сообщение:
//блокирует RLock для поиска соединения
//если соединение не найдено — возвращает ошибку.
//блокирует мьютекс wrapper.mutex.Lock() для этого соединения (гарантирует, что при отправке не будет конкуренции)
//вызывает WriteJSON для отправки сообщения.
//После отправки освобождает мьютекс.
func (cm *ConnectionManager) SendMessage(id string, message contracts.WSMessage) error {
	cm.mutex.RLock()
	wrapper, exists := cm.connections[id]
	cm.mutex.RUnlock()

	if !exists {
		return ErrConnectionNotFound
	}

	wrapper.mutex.Lock()
	defer wrapper.mutex.Unlock()

	return wrapper.conn.WriteJSON(message)
}