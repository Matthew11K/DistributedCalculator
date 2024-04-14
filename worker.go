package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/lib/pq"
	"github.com/streadway/amqp"
	"log"
	"math/big"
	"os"
	"strconv"
)

// Функция для вычисления факториала числа
func factorial(n int) (*big.Int, error) {
	if n < 0 {
		return nil, fmt.Errorf("negative number")
	}
	result := big.NewInt(1)
	for i := 2; i <= n; i++ {
		result.Mul(result, big.NewInt(int64(i)))
	}
	return result, nil
}

// Функция для вычисления числа Фибоначчи
func fibonacci(n int) (int, error) {
	if n <= 0 {
		return 0, fmt.Errorf("negative number or zero")
	}
	if n <= 2 {
		return 1, nil
	}
	a, b := 1, 1
	for i := 3; i <= n; i++ {
		a, b = b, a+b
	}
	return b, nil
}

// Определяем структуру Task для декодирования JSON сообщения.
type Task struct {
	ID        int      `json:"id"`
	Operation string   `json:"operation"`
	Data      []string `json:"data"` // Данные ожидаются как массив строк
}

// Функция performArithmeticOperation обновлена для работы с массивом строк.
func performArithmeticOperation(operation string, numbers []string) (string, error) {
	if len(numbers) != 2 {
		return "", fmt.Errorf("expected two numbers for operation, got %d", len(numbers))
	}
	num1, err := strconv.ParseFloat(numbers[0], 64)
	if err != nil {
		return "", fmt.Errorf("error parsing first number: %v", err)
	}
	num2, err := strconv.ParseFloat(numbers[1], 64)
	if err != nil {
		return "", fmt.Errorf("error parsing second number: %v", err)
	}
	var result float64
	switch operation {
	case "add":
		result = num1 + num2
	case "subtract":
		result = num1 - num2
	case "multiply":
		result = num1 * num2
	case "divide":
		if num2 == 0 {
			return "", fmt.Errorf("division by zero")
		}
		result = num1 / num2
	default:
		return "", fmt.Errorf("unknown operation %s", operation)
	}
	return fmt.Sprintf("%f", result), nil
}

// Обработчик сообщений из очереди RabbitMQ.
func handleMessage(d amqp.Delivery) {
	log.Printf("Received raw message: %s", string(d.Body))
	var task Task
	if err := json.Unmarshal(d.Body, &task); err != nil {
		log.Printf("Error decoding task message: %v", err)
		return
	}

	var result string
	var err error

	// Вызов соответствующей функции в зависимости от операции.
	switch task.Operation {
	case "add", "subtract", "multiply", "divide":
		result, err = performArithmeticOperation(task.Operation, task.Data)
	case "factorial":
		// Обработка факториала.
		num, err := strconv.Atoi(task.Data[0]) // Предполагается, что Data содержит только один элемент.
		if err != nil {
			log.Printf("Error converting data to int for factorial: %v", err)
			return
		}
		factorialResult, err := factorial(num)
		if err != nil {
			log.Printf("Error calculating factorial: %v", err)
			return
		}
		result = factorialResult.String()
	case "fibonacci":
		// Обработка числа Фибоначчи.
		num, err := strconv.Atoi(task.Data[0]) // Предполагается, что Data содержит только один элемент.
		if err != nil {
			log.Printf("Error converting data to int for fibonacci: %v", err)
			return
		}
		fibonacciResult, err := fibonacci(num)
		if err != nil {
			log.Printf("Error calculating fibonacci: %v", err)
			return
		}
		result = strconv.Itoa(fibonacciResult)
	default:
		log.Printf("Unknown operation: %s", task.Operation)
		return
	}

	if err != nil {
		log.Printf("Error processing task ID %d: %v", task.ID, err)
		result = "error"
	}

	// Обновление статуса и результата задачи в базе данных.
	_, err = db.Exec("UPDATE tasks_new SET status = $1, result = $2 WHERE id = $3", "completed", result, task.ID)
	if err != nil {
		log.Printf("Failed to update task ID %d with result: %v", task.ID, err)
		return
	}

	log.Printf("Task ID %d completed with result: %s", task.ID, result)
}

var db *sql.DB

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
		panic(msg)
	}
}

func main() {
	// Подключение к RabbitMQ
	conn, err := amqp.Dial(os.Getenv("AMQP_URL"))
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"tasks_queue", // name
		true,          // durable
		false,         // delete when unused
		false,         // exclusive
		false,         // no-wait
		nil,           // arguments
	)
	failOnError(err, "Failed to declare a queue")

	// Подключение к PostgreSQL
	db, err = sql.Open("postgres", os.Getenv("DATABASE_URL"))
	failOnError(err, "Failed to connect to PostgreSQL")
	defer db.Close()

	err = db.Ping()
	failOnError(err, "Failed to ping PostgreSQL")

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	failOnError(err, "Failed to register a consumer")

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			handleMessage(d)
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}
