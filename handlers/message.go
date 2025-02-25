package handlers

import (
	"database/sql"
	"fmt"
	"log"
	"sync"

	"bot/database"
	"bot/keyboards"
	"bot/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var (
	creatingOrder   sync.Map
	waitingQuantity sync.Map
	currentOrders   sync.Map
)

func StartBot(bot *tgbotapi.BotAPI, db *sql.DB) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil {
			HandleMessage(bot, update.Message, db)
		} else if update.CallbackQuery != nil {
			HandleCallbackQuery(bot, update.CallbackQuery, db)
		}
	}
}

func HandleMessage(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {

	// Проверяем, что сообщение пришло из личного чата
	if message.Chat.Type != "private" {
		return // Игнорируем сообщения из групп и каналов
	}

	userID := message.From.ID // Добавляем определение userID
	text := message.Text

	switch {
	case text == "/start":
		msg := tgbotapi.NewMessage(message.Chat.ID, "Добро пожаловать, коллега! Здесь Вы можете создать заявку или посмотреть свои заказы.")
		msg.ReplyMarkup = keyboards.GetStartKeyboard()
		bot.Send(msg)

	case text == "Создать заявку на покупку":
		HandleCreateOrder(bot, message.Chat.ID, userID)

	case text == "Мои заказы":
		HandleMyOrders(bot, message.Chat.ID, userID, db)

	case text == "Завершить заявку":
		HandleCompleteOrder(bot, message.Chat.ID, userID, message.From.UserName, db)
		/*if orders, ok := currentOrders.Load(userID); !ok || len(orders.([]models.Order)) == 0 {
			msg := tgbotapi.NewMessage(message.Chat.ID, "Вы не добавили ни одного товара в заявку.")
			bot.Send(msg)
			return
		}

		// Сохраняем заявку
		if orders, ok := currentOrders.Load(userID); ok {
			err := database.SaveOrderToDatabase(bot, userID, message.From.UserName, orders.([]models.Order))
			if err != nil {
				log.Printf("Ошибка сохранения заказа: %v\n", err)
				msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка при сохранении заказа. Попробуйте позже.")
				bot.Send(msg)
				return
			}
		}

		// Очищаем данные
		creatingOrder.Delete(userID)
		currentOrders.Delete(userID)
		waitingQuantity.Delete(userID)

		msg := tgbotapi.NewMessage(message.Chat.ID, "Заявка успешно сохранена!")
		msg.ReplyMarkup = keyboards.GetStartKeyboard()
		bot.Send(msg)*/

	case text == "Отмена":
		HandleCancelOrder(bot, message.Chat.ID, userID)
		/*if value, ok := creatingOrder.Load(userID); ok && value.(bool) {
			creatingOrder.Delete(userID)
			currentOrders.Delete(userID)
			waitingQuantity.Delete(userID)

			msg := tgbotapi.NewMessage(message.Chat.ID, "Создание заявки отменено.")
			msg.ReplyMarkup = keyboards.GetStartKeyboard()
			bot.Send(msg)
		} else {
			msg := tgbotapi.NewMessage(message.Chat.ID, "Нечего отменять.")
			bot.Send(msg)
		}*/

	case text == "/allorders" || text == "Все заказы" || text == "все заказы" || text == "все заявки" || text == "все Заявки":
		ShowOrders(bot, message, "SELECT requests.id, requests.user_id, specialists.name as username, date, is_completed FROM requests join specialists on requests.user_id = specialists.user_id;")

	case text == "/unfinished" || text == "Невыполненные" || text == "невыполненные":
		ShowOrders(bot, message, "SELECT requests.id, requests.user_id, specialists.name as username, date, is_completed FROM requests join specialists on requests.user_id = specialists.user_id WHERE !is_completed;")

	case text == "/today" || text == "Сегодня" || text == "сегодня":
		ShowOrders(bot, message, "SELECT requests.id, requests.user_id, specialists.name as username, date, is_completed FROM requests join specialists on requests.user_id = specialists.user_id WHERE date(date)=CURDATE();")

	default:
		// Проверяем, находится ли пользователь в процессе создания заказа
		if creatingOrderValue, ok := creatingOrder.Load(userID); ok && creatingOrderValue.(bool) {
			HandleOrderCreation(bot, message)
		} else if userNames, err := database.UserName(text); userNames != nil && err == nil {
			//проверяем нет ли введённых имен пользователей
			for _, name := range userNames {
				msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("Есть такой: %s.", name))
				bot.Send(msg)
			}
		} else {
			// Создаем сообщение с инлайн-кнопками
			msg := tgbotapi.NewMessage(message.Chat.ID, "Неизвестная команда. Вы хотите создать заказ или посмотреть статус своих заказов?")
			msg.ReplyMarkup = keyboards.GetUnknownCommandKeyboard() // Добавляем инлайн-клавиатуру
			bot.Send(msg)

			// Логируем неизвестную команду
			log.Printf("Неизвестная команда от пользователя %d: %s\n", userID, text)
		}
	}
}

// HandleOrderCreation обрабатывает создание заказа
func HandleOrderCreation(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	userID := message.From.ID
	text := message.Text

	if quantity, _ := waitingQuantity.Load(userID); quantity != nil {
		quantity := text
		// Обновляем количество для последнего товара
		if orders, ok := currentOrders.Load(userID); ok {
			orderList := orders.([]models.Order)
			orderList[len(orderList)-1].Quantity = quantity
			currentOrders.Store(userID, orderList)
		}

		waitingQuantity.Delete(userID)
		msg := tgbotapi.NewMessage(message.Chat.ID, "Товар добавлен в заявку. Можете ввести следующий товар или завершить создание заявки.")
		msg.ReplyMarkup = keyboards.GetOrderActionsKeyboard() // Добавляем инлайн-клавиатуру
		bot.Send(msg)
	} else {
		// Обрабатываем ввод названия товара
		if orders, ok := currentOrders.Load(userID); ok {
			orderList := orders.([]models.Order)
			orderList = append(orderList, models.Order{ProductName: text})
			currentOrders.Store(userID, orderList)
		} else {
			currentOrders.Store(userID, []models.Order{{ProductName: text}})
		}

		waitingQuantity.Store(userID, true)
		msg := tgbotapi.NewMessage(message.Chat.ID, "Введите количество для данного товара.")
		bot.Send(msg)
	}
}

// SendGroupChatMessage отправляет сообщение с кнопкой в групповой чат
func SendGroupChatMessage(bot *tgbotapi.BotAPI, chatID int64) {
	// Создаем inline-кнопку
	button := tgbotapi.NewInlineKeyboardButtonURL("Перейти к боту", "https://t.me/asset_managment_bot")
	markup := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(button))

	// Создаем сообщение
	msg := tgbotapi.NewMessage(chatID, "Нажмите кнопку, чтобы перейти в чат, где можно сделать заказ запасных частей, инструментов и принадлежностей")
	msg.ReplyMarkup = markup

	// Отправляем сообщение
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Ошибка отправки сообщения в групповой чат: %v\n", err)
	}
}

// ShowOrders отрабатывает SQL запрос сегодя, невыполненные, все заказы
func ShowOrders(bot *tgbotapi.BotAPI, message *tgbotapi.Message, SQLquery string) {

	userID := message.From.ID // Добавляем определение userID

	// Проверяем, является ли пользователь админом
	isAdmin, err := database.IsAdmin(userID)
	if err != nil {
		log.Printf("Ошибка проверки администратора: %v\n", err)
		return
	}

	if !isAdmin {
		msg := tgbotapi.NewMessage(message.Chat.ID, "У вас нет прав для выполнения этой команды.")
		bot.Send(msg)
		return
	}

	// Получаем все заявки запросом присланным при вызове функции
	orders, err := database.GetOrders(SQLquery)
	if err != nil {
		log.Printf("Ошибка получения заявок: %v\n", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "Ошибка при получении заявок.")
		bot.Send(msg)
		return
	}

	if len(orders) == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Нет доступных заявок.")
		bot.Send(msg)
		return
	}

	// Отправляем список заявок
	for _, order := range orders {
		response := fmt.Sprintf("Заявка #%d\n", order.ID)
		response += fmt.Sprintf("Пользователь: %s\n", order.Username)
		response += fmt.Sprintf("Статус: %s\n", order.Status)
		response += "Позиции заказа:\n"
		for _, item := range order.Items {
			response += fmt.Sprintf("- %s: %s \n", item.ProductName, item.Quantity)
		}

		// Добавляем кнопку "Отметить как выполненную"
		markup := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Отметить как выполненную", fmt.Sprintf("complete_%d", order.ID)),
			),
		)

		msg := tgbotapi.NewMessage(message.Chat.ID, response)

		if order.Status == "Не выполнен" {
			msg.ReplyMarkup = markup
		}
		bot.Send(msg)
	}

}

// HandleCreateOrder обрабатывает команду "Создать заявку на покупку"
func HandleCreateOrder(bot *tgbotapi.BotAPI, chatID int64, userID int64) {
	creatingOrder.Store(userID, true)
	currentOrders.Store(userID, []models.Order{})
	msg := tgbotapi.NewMessage(chatID, "Введите название продукта для добавления в заявку.")
	msg.ReplyMarkup = keyboards.GetOrderActionsKeyboard() // Добавляем инлайн-клавиатуру
	msg.ReplyMarkup = keyboards.GetOrderCreationKeyboard()
	bot.Send(msg)
}

// HandleMyOrders обрабатывает команду "Мои заказы"
func HandleMyOrders(bot *tgbotapi.BotAPI, chatID int64, userID int64, db *sql.DB) {
	orders, err := database.GetUserOrders(userID)
	if err != nil {
		log.Printf("Ошибка получения заказов: %v", err)
		msg := tgbotapi.NewMessage(chatID, "Произошла ошибка при получении заказов. Попробуйте позже.")
		bot.Send(msg)
		return
	}

	if len(orders) == 0 {
		msg := tgbotapi.NewMessage(chatID, "У вас пока нет созданных заявок.")
		bot.Send(msg)
		return
	}

	msg := tgbotapi.NewMessage(chatID, "Ваши заказы. Выберите один для просмотра подробностей:")
	msg.ReplyMarkup = keyboards.GetUserOrdersInlineKeyboard(userID)
	bot.Send(msg)
}

func HandleCompleteOrder(bot *tgbotapi.BotAPI, chatID int64, userID int64, username string, db *sql.DB) {
	// Проверяем, есть ли товары в заявке
	if orders, ok := currentOrders.Load(userID); !ok || len(orders.([]models.Order)) == 0 {
		msg := tgbotapi.NewMessage(chatID, "Вы не добавили ни одного товара в заявку.")
		bot.Send(msg)
		return
	}

	// Сохраняем заявку
	if orders, ok := currentOrders.Load(userID); ok {
		err := database.SaveOrderToDatabase(bot, userID, username, orders.([]models.Order))
		if err != nil {
			log.Printf("Ошибка сохранения заказа: %v", err)
			msg := tgbotapi.NewMessage(chatID, "Произошла ошибка при сохранении заказа. Попробуйте позже.")
			bot.Send(msg)
			return
		}
	}

	// Очищаем данные
	creatingOrder.Delete(userID)
	currentOrders.Delete(userID)
	waitingQuantity.Delete(userID)

	msg := tgbotapi.NewMessage(chatID, "Заявка успешно сохранена!")
	msg.ReplyMarkup = keyboards.GetStartKeyboard()
	bot.Send(msg)
}

func HandleCancelOrder(bot *tgbotapi.BotAPI, chatID int64, userID int64) {
	if value, ok := creatingOrder.Load(userID); ok && value.(bool) {
		// Если пользователь находится в процессе создания заказа, отменяем его
		creatingOrder.Delete(userID)
		currentOrders.Delete(userID)
		waitingQuantity.Delete(userID)

		msg := tgbotapi.NewMessage(chatID, "Создание заявки отменено.")
		msg.ReplyMarkup = keyboards.GetStartKeyboard()
		bot.Send(msg)
	} else {
		// Если заявка не создается, сообщаем, что отменять нечего
		msg := tgbotapi.NewMessage(chatID, "Нечего отменять.")
		bot.Send(msg)
	}
}
