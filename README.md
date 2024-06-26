# Распределенный калькулятор с микросервисной архитектурой

Проект представляет собой распределенный калькулятор с использованием 
микросервисной архитектуры. Он включает в себя клиентское веб-приложение 
для отправки вычислительных задач, серверную часть для обработки этих 
задач, а также механизм обмена сообщениями с использованием RabbitMQ.

## Клиентское веб-приложение

Клиентское веб-приложение позволяет пользователям отправлять запросы на 
вычисление арифметических операций (сложение, вычитание, умножение, 
деление), факториала и чисел Фибоначчи. Они отправляют свои запросы через 
HTTP на сервер, который принимает их, добавляет в очередь RabbitMQ для 
обработки и записывает в базу данных PostgreSQL информацию о задаче.

## Серверная часть

Серверная часть приложения слушает очередь RabbitMQ на предмет новых 
задач, обрабатывает их, используя параллельные вычисления, и записывает 
результаты в базу данных. Затем клиент может запросить результаты задачи, 
используя уникальный идентификатор задачи.

## Используемые технологии

- Язык программирования Go для серверной части
- База данных PostgreSQL для хранения информации о задачах
- Библиотеки `gorilla/mux` и стандартный пакет `database/sql` для работы с 
HTTP и SQL соответственно
- Библиотеки `streadway/amqp` и `encoding/json` для взаимодействия с 
RabbitMQ и обработки JSON

## Ключевые функции

Ключевые функции проекта включают в себя обработку арифметических 
операций, факториала и чисел Фибоначчи, а также механизм обработки 
сообщений через RabbitMQ с использованием параллельных вычислений для 
улучшения производительности.

## Начало работы
1. Клонируйте репозиторий. 
2. Перейдите в папку проекта: `cd yourproject`
3. Скопируйте `.env.example` в `.env` и настройте переменные окружения в файле `.env`.
4. Запустите проект: `docker-compose up --build`
5. Проверьте работу сервера, обратившись к `http://localhost:8080`.

## API
### Отправка задач на вычисление

Для отправки вычислительных задач используйте следующий API-метод:

**POST /calculate**
- **Content-Type**: `application/json`
- **Body**: JSON объект, содержащий поля `operation` (операция для выполнения) и `data` (данные для операции, например, числа для арифметических операций).

Пример запроса на сложение двух чисел:
```json
{
  "operation": "add",
  "data": ["5", "3"]
}
```
Ответ будет содержать HTTP статус 202 (Accepted) и ID задачи, если запрос был успешно обработан и задача добавлена в очередь.

### Получение результатов вычислений

Для получения результатов задачи используйте следующий API-метод:

**GET /result/{taskId}**
- **Path variable**: `taskId` - уникальный идентификатор задачи, полученный при отправке.

Пример запроса:
```plaintext
GET /result/123
```
