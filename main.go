package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/streadway/amqp"
	"io"
	"log"
	"net/http"
	"os"
)

var db *sql.DB
var ch *amqp.Channel // Глобальная переменная для канала RabbitMQ

func main() {
	var err error // Переменная для ошибок

	// Инициализация соединения с базой данных PostgreSQL
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("POSTGRES_HOST"),
		os.Getenv("POSTGRES_PORT"),
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRES_DB"))
	db, err = sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatalf("Ошибка при подключении к базе данных: %v", err)
	}
	defer db.Close()

	// Проверка соединения с базой данных
	err = db.Ping()
	if err != nil {
		log.Fatalf("Не удалось установить соединение с базой данных: %v", err)
	}

	// Инициализация соединения с RabbitMQ
	conn, err := amqp.Dial(os.Getenv("AMQP_URL"))
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	ch, err = conn.Channel() // Инициализация глобальной переменной для канала RabbitMQ
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}
	defer ch.Close()

	// Объявление очереди для задач
	_, err = ch.QueueDeclare(
		"tasks_queue", // name
		true,          // durable
		false,         // delete when unused
		false,         // exclusive
		false,         // no-wait
		nil,           // arguments
	)
	if err != nil {
		log.Fatalf("Failed to declare a queue: %v", err)
	}

	// Инициализация маршрутизатора и регистрация обработчиков
	r := mux.NewRouter()
	r.HandleFunc("/calculate", calculateHandler).Methods("POST")
	r.HandleFunc("/result/{taskId}", resultHandler).Methods("GET")

	// Запуск HTTP сервера
	log.Println("Сервер запущен на порту 8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}

func calculateHandler(w http.ResponseWriter, r *http.Request) {
	var input map[string]interface{}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	// Декодирование полученного JSON
	err = json.Unmarshal(body, &input)
	if err != nil {
		log.Printf("Ошибка при декодировании запроса: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// Вставка задачи в базу данных и получение ID
	var taskId int
	err = db.QueryRow("INSERT INTO tasks_new (status) VALUES ($1) RETURNING id", "received").Scan(&taskId)
	if err != nil {
		log.Printf("Ошибка при вставке задачи: %v", err) // Логируем саму ошибку
		http.Error(w, "Failed to insert task into database", http.StatusInternalServerError)
		return
	}
	log.Printf("Task successfully inserted with ID: %d", taskId) // Подтверждение успешной вставки

	// Формирование JSON объекта для отправки в очередь
	taskData := map[string]interface{}{
		"id":        taskId,
		"operation": input["operation"],
		"data":      input["data"],
	}
	taskDataBytes, err := json.Marshal(taskData)
	if err != nil {
		log.Printf("Ошибка при сериализации данных задачи: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	log.Printf("Publishing message: %s", taskDataBytes)

	// Отправка задачи в очередь RabbitMQ
	err = ch.Publish(
		"",
		"tasks_queue",
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        taskDataBytes,
		})
	if err != nil {
		http.Error(w, "Failed to send to RabbitMQ", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(fmt.Sprintf("Task received and accepted for processing with ID: %d\n", taskId)))
}

func resultHandler(w http.ResponseWriter, r *http.Request) {
	// Получение ID задачи из пути запроса
	vars := mux.Vars(r)
	taskId := vars["taskId"]

	// Поиск результата задачи в базе данных
	var status, result string
	err := db.QueryRow("SELECT status, result FROM tasks_new WHERE id = $1", taskId).Scan(&status, &result)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "No task found with given ID", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to retrieve task", http.StatusInternalServerError)
		}
		return
	}

	// Ответ клиенту с состоянием и результатом задачи
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("Status: %s, Result: %s", status, result)))
}
