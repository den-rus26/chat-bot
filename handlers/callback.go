package handlers

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"bot/database"
	"database/sql"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func HandleCallbackQuery(bot *tgbotapi.BotAPI, query *tgbotapi.CallbackQuery, db *sql.DB) {

	// Проверяем, что callback пришел из личного чата
	if query.Message.Chat.Type != "private" {
		return // Игнорируем callback-запросы из групп и каналов
	}

	data := query.Data
	log.Printf("Получен callback запрос: %s\n", data)

	switch data {
	case "create_order":
		// Обработка нажатия на кнопку "Создать заказ"
		HandleCreateOrder(bot, query.Message.Chat.ID, query.From.ID)

	case "my_orders":
		// Обработка нажатия на кнопку "Мои заказы"
		HandleMyOrders(bot, query.Message.Chat.ID, query.From.ID, db)

	case "complete_order":
		// Обработка нажатия на кнопку "Завершить заказ"
		HandleCompleteOrder(bot, query.Message.Chat.ID, query.From.ID, query.From.UserName, db)

	case "cancel_order":
		// Обработка нажатия на кнопку "Завершить заказ"
		HandleCancelOrder(bot, query.Message.Chat.ID, query.From.ID)

	default:
		if strings.HasPrefix(data, "order_") {
			orderIDStr := strings.TrimPrefix(data, "order_")
			orderID, err := strconv.Atoi(orderIDStr)
			if err != nil {
				log.Printf("Ошибка обработки ID заказа: %v\n", err)
				return
			}

			// Используем GetOrderDetails из пакета database
			orderDetails, err := database.GetOrderDetails(orderID)
			if err != nil {
				msg := tgbotapi.NewMessage(query.Message.Chat.ID, "Ошибка получения данных о заказе. Попробуйте позже.")
				bot.Send(msg)
				return
			}

			response := fmt.Sprintf("Подробности заказа #%d:\n", orderDetails.ID)
			response += fmt.Sprintf("Статус: %s\n", orderDetails.Status)
			response += "Позиции заказа:\n"
			for _, item := range orderDetails.Items {
				response += fmt.Sprintf("- %s: %s \n", item.ProductName, item.Quantity)
			}

			msg := tgbotapi.NewMessage(query.Message.Chat.ID, response)
			bot.Send(msg)
		} else if strings.HasPrefix(data, "complete_") {
			// Админ отметил заявку как выполненную
			orderIDStr := strings.TrimPrefix(data, "complete_")
			orderID, err := strconv.Atoi(orderIDStr)
			if err != nil {
				log.Printf("Ошибка обработки ID заказа: %v\n", err)
				return
			}

			// Отмечаем заявку как выполненную
			if err := database.MarkOrderAsCompleted(orderID); err != nil {
				log.Printf("Ошибка отметки заявки как выполненной: %v\n", err)
				msg := tgbotapi.NewMessage(query.Message.Chat.ID, "Ошибка при обновлении заявки.")
				bot.Send(msg)
				return
			}

			// Оповещаем пользователя
			database.NotifyUser(bot, orderID)

			msg := tgbotapi.NewMessage(query.Message.Chat.ID, fmt.Sprintf("Заявка #%d отмечена как выполненная.", orderID))
			bot.Send(msg)
		}
	}
}
