package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	DBDSN            string
	TelegramBotToken string
	GroupChatID      int64
}

func LoadConfig() (*Config, error) {
	dbDSN := os.Getenv("DB_DSN")
	if dbDSN == "" {
		return nil, fmt.Errorf("отсутствует DB_DSN в .env")
	}

	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		return nil, fmt.Errorf("отсутствует TELEGRAM_BOT_TOKEN в .env")
	}

	botGroupChatID := os.Getenv("GROUPCHATID")
	if botGroupChatID == "" {
		return nil, fmt.Errorf("отсутствует GROUPCHATID в .env")
	}

	GroupChatID, err := strconv.ParseInt(botGroupChatID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("ошибка формата группового чата")
	}

	return &Config{
		DBDSN:            dbDSN,
		TelegramBotToken: botToken,
		GroupChatID:      GroupChatID,
	}, nil
}
