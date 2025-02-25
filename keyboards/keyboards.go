package keyboards

import (
	"bot/database"
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// GetStartKeyboard возвращает основную клавиатуру
func GetStartKeyboard() tgbotapi.ReplyKeyboardMarkup {
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Создать заявку на покупку"),
			tgbotapi.NewKeyboardButton("Мои заказы"),
		),
	)
	// Устанавливаем OneTimeKeyboard в false, чтобы клавиатура не скрывалась после нажатия
	keyboard.OneTimeKeyboard = false

	return keyboard
}

// GetOrderCreationKeyboard возвращает клавиатуру для создания заказа
func GetOrderCreationKeyboard() tgbotapi.ReplyKeyboardMarkup {
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Завершить заявку"),
			tgbotapi.NewKeyboardButton("Отмена"),
		),
	)
	// Устанавливаем OneTimeKeyboard в false, чтобы клавиатура не скрывалась после нажатия
	keyboard.OneTimeKeyboard = false

	return keyboard

}

// GetUserOrdersInlineKeyboard создает inline-клавиатуру для списка заказов
func GetUserOrdersInlineKeyboard(userID int64) tgbotapi.InlineKeyboardMarkup {
	orders, err := database.GetUserOrders(userID)
	if err != nil {
		log.Printf("Ошибка получения заказов: %v\n", err)
		return tgbotapi.NewInlineKeyboardMarkup() // Возвращаем пустую клавиатуру в случае ошибки
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	for _, order := range orders {
		parts := strings.Split(order, " | ")
		if len(parts) > 0 {
			orderID := strings.TrimPrefix(parts[0], "#")
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(order, "order_"+orderID),
			))
		}
	}

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func GetUnknownCommandKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Создать заявку", "create_order"),
			tgbotapi.NewInlineKeyboardButtonData("Мои заказы", "my_orders"),
		),
	)
}

func GetOrderActionsKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Завершить заявку", "complete_order"),
			tgbotapi.NewInlineKeyboardButtonData("Передумал/Отмена", "cancel_order"),
		),
	)
}
