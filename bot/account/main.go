package account

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {

}

func ShowAccountInfo(chatID int64, userID int64) tgbotapi.MessageConfig {
	msg := tgbotapi.NewMessage(chatID, "Ваш профиль:\nID пользователя: "+fmt.Sprint(userID)+"\nБаланс: 1000 монет")
	return msg
}
