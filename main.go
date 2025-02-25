package main

import (
	"bot/config"
	"bot/database"
	"bot/handlers"
	"log"
	"os"
	"path/filepath"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/debug"
	"golang.org/x/sys/windows/svc/eventlog"
)

var elog debug.Log

type myService struct{}

func (m *myService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	// Сообщаем SCM, что служба запускается
	changes <- svc.Status{State: svc.StartPending}

	// Инициализация
	log.Println("Инициализация программы...")
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}

	db, err := database.InitDB(cfg.DBDSN)
	if err != nil {
		log.Fatalf("Ошибка инициализации базы данных: %v", err)
	}
	defer db.Close()

	bot, err := tgbotapi.NewBotAPI(cfg.TelegramBotToken)
	if err != nil {
		log.Fatalf("Ошибка инициализации бота: %v", err)
	}

	// Сообщаем SCM, что служба запущена
	changes <- svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}

	// Отправка сообщения в группу
	//handlers.SendGroupChatMessage(bot, cfg.GroupChatID)

	// Запуск обработчиков в отдельной горутине
	go handlers.StartBot(bot, db)

	// Основной цикл обработки команд SCM
	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				// Отправляем текущий статус
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				// Сообщаем SCM, что служба останавливается
				changes <- svc.Status{State: svc.StopPending}
				log.Println("Служба останавливается...\n")
				return
			default:
				log.Printf("Неизвестный запрос: %v\n", c)
			}
		}
	}
}

func main() {
	// Логирование в файл
	dir := "D:/bot/"
	logFile := filepath.Join(dir, "program.log")
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Ошибка открытия файла логов: %v\n", err)
	}
	defer file.Close()
	log.SetOutput(file)

	// Загрузка .env
	envPath := filepath.Join(dir, ".env")
	if err := godotenv.Load(envPath); err != nil {
		log.Fatal("Отсутствует .env файл")
	}

	// Инициализация логов службы
	var errLog error
	elog, errLog = eventlog.Open("MyService")
	if errLog != nil {
		log.Fatalf("Ошибка открытия лога: %v\n", errLog)
	}
	defer elog.Close()

	elog.Info(1, "Запуск службы...")

	// Запуск службы
	err = svc.Run("MyService", &myService{})
	if err != nil {
		elog.Error(1, "Ошибка запуска службы: "+err.Error())
		return
	}

	elog.Info(1, "Служба остановлена.")
}
