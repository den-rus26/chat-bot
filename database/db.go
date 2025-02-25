package database

import (
	"bot/models"
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var db *sql.DB

// InitDB инициализирует подключение к базе данных
func InitDB(dsn string) (*sql.DB, error) {
	var err error
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("ошибка подключения к базе данных: %w", err)
	}

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("ошибка проверки соединения с базой данных: %w", err)
	}

	return db, nil
}

// GetDB возвращает текущее подключение к базе данных
func GetDB() *sql.DB {
	return db
}

// IsAdmin проверяет, является ли пользователь администратором
func IsAdmin(userID int64) (bool, error) {
	db := GetDB()
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM admins WHERE user_id = ?)", userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("ошибка проверки администратора: %w", err)
	}
	return exists, nil
}

// Унифицированная функция выборки заказов
func GetOrders(SQLquery string) ([]models.OrderDetails, error) {
	db := GetDB()
	var orders []models.OrderDetails

	//SQLquery заранее сформарованный SQL запрос
	rows, err := db.Query(SQLquery)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения всех заявок: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var order models.OrderDetails
		var userID int64
		var username string
		var date string
		var isCompleted bool

		if err := rows.Scan(&order.ID, &userID, &username, &date, &isCompleted); err != nil {
			log.Printf("Ошибка сканирования заявки: %v\n", err)
			continue
		}

		order.Username = username

		order.Status = "Не выполнен"
		if isCompleted {
			order.Status = "Выполнен"
		}

		// Получаем товары для заявки
		items, err := getOrderItems(order.ID)
		if err != nil {
			log.Printf("Ошибка получения товаров для заявки #%d: %v\n", order.ID, err)
			continue
		}
		order.Items = items

		orders = append(orders, order)
	}

	return orders, nil
}

// getOrderItems возвращает товары для конкретной заявки
func getOrderItems(orderID int) ([]models.Order, error) {
	db := GetDB()
	var items []models.Order

	rows, err := db.Query("SELECT product_name, quantity FROM request_items WHERE request_id = ?", orderID)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения товаров: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item models.Order
		if err := rows.Scan(&item.ProductName, &item.Quantity); err != nil {
			log.Printf("Ошибка сканирования товара: %v\n", err)
			continue
		}
		items = append(items, item)
	}

	return items, nil
}

// GetUserOrders возвращает список заказов пользователя
func GetUserOrders(userID int64) ([]string, error) {
	db := GetDB()
	rows, err := db.Query("SELECT id, date, is_completed FROM requests WHERE user_id = ?", userID)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения заявок пользователя: %w", err)
	}
	defer rows.Close()

	var orders []string
	for rows.Next() {
		var id int
		var date string
		var isCompleted bool
		if err := rows.Scan(&id, &date, &isCompleted); err != nil {
			log.Printf("Ошибка сканирования строки: %v\n", err)
			continue
		}
		status := "Не выполнена"
		if isCompleted {
			status = "Выполнена"
		}
		orders = append(orders, fmt.Sprintf("#%d | Дата: %s | Статус: %s", id, date, status))
	}

	return orders, nil
}

// GetOrderDetails возвращает детали заказа по его ID
func GetOrderDetails(orderID int) (models.OrderDetails, error) {
	var details models.OrderDetails

	var isCompleted bool
	err := db.QueryRow("SELECT is_completed FROM requests WHERE id = ?", orderID).Scan(&isCompleted)
	if err != nil {
		return details, err
	}
	details.ID = orderID
	details.Status = "Не выполнен"
	if isCompleted {
		details.Status = "Выполнен"
	}

	rows, err := db.Query("SELECT product_name, quantity FROM request_items WHERE request_id = ?", orderID)
	if err != nil {
		return details, err
	}
	defer rows.Close()

	for rows.Next() {
		var item models.Order
		if err := rows.Scan(&item.ProductName, &item.Quantity); err != nil {
			log.Printf("Ошибка чтения позиции заказа: %v\n", err)
			continue
		}
		details.Items = append(details.Items, item)
	}

	return details, nil
}

// SaveOrderToDatabase сохраняет заказ в базе данных
func SaveOrderToDatabase(bot *tgbotapi.BotAPI, userID int64, username string, orders []models.Order) error {
	db := GetDB()

	// Начинаем транзакцию
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("ошибка начала транзакции: %w", err)
	}

	// Откатываем транзакцию при ошибке
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// Вставляем заявку в таблицу requests
	result, err := tx.Exec("INSERT INTO requests (user_id, username, date) VALUES (?, ?, NOW())", userID, username)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("ошибка сохранения заказа: %w", err)
	}

	// Получаем ID вставленной заявки
	requestID, err := result.LastInsertId()
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("ошибка получения ID заявки: %w", err)
	}

	// Вставляем товары в таблицу request_items
	for _, order := range orders {
		_, err := tx.Exec("INSERT INTO request_items (request_id, product_name, quantity) VALUES (?, ?, ?)", requestID, order.ProductName, order.Quantity)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("ошибка сохранения позиции заказа: %w", err)
		}
	}

	// Коммитим транзакцию
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("ошибка коммита транзакции: %w", err)
	}

	// Оповещаем админов
	NotifyAdmins(bot, int(requestID), username)

	return nil
}

// MarkOrderAsCompleted отмечает заявку как выполненную
func MarkOrderAsCompleted(orderID int) error {
	db := GetDB()
	_, err := db.Exec("UPDATE requests SET is_completed = TRUE, date_done = NOW() WHERE id = ?", orderID)
	if err != nil {
		return fmt.Errorf("ошибка обновления заявки: %w", err)
	}

	return nil
}

func NotifyUser(bot *tgbotapi.BotAPI, orderID int) {
	db := GetDB()
	rows, err := db.Query("SELECT user_id FROM requests WHERE id = ?", orderID)
	if err != nil {
		log.Printf("ошибка получения данных для сообщения пользователю о выполненной заявке: %v\n", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var messageID int64
		if err := rows.Scan(&messageID); err != nil {
			log.Printf("Ошибка сканирования ID выполненого заказа: %v\n", err)
			continue
		}
		msg := tgbotapi.NewMessage(messageID, fmt.Sprintf("Ваша заявка, номер: #%d выполнена.", orderID))
		bot.Send(msg)
	}

}

func NotifyAdmins(bot *tgbotapi.BotAPI, orderID int, username string) {
	db := GetDB()
	rows, err := db.Query("SELECT user_id FROM admins")
	if err != nil {
		log.Printf("Ошибка получения списка администраторов: %v\n", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var adminID int64
		if err := rows.Scan(&adminID); err != nil {
			log.Printf("Ошибка сканирования ID администратора: %v\n", err)
			continue
		}

		rows, err := db.Query("SELECT specialists.name FROM specialists JOIN requests on specialists.user_id = requests.user_id WHERE requests.id = ?", orderID)
		if err != nil {
			log.Printf("Ошибка получения имени пользователя: %v\n", err)
			return
		}
		defer rows.Close()
		for rows.Next() {
			var name string

			if err := rows.Scan(&name); err != nil {
				log.Printf("Ошибка сканирования имени пользователя %v\n", err)
				continue
			}

			msg := tgbotapi.NewMessage(adminID, fmt.Sprintf("Новая заявка №%d от пользователя %s", orderID, name))
			bot.Send(msg)
		}
	}
}

func UserName(userName string) ([]string, error) {
	db := GetDB()
	rows, err := db.Query("SELECT name FROM specialists WHERE name LIKE concat('%', ?, '%')", userName)
	if err != nil {
		log.Printf("ошибка поиска пользователя по имени: %v\n", err)
		return nil, err
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			log.Printf("Ошибка сканирования имён: %v\n", err)
			continue
		}
		names = append(names, name)
	}
	return names, nil
}
