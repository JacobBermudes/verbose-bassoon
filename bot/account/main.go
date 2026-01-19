package account

import (
	"encoding/json"
	"fmt"
	"net/http"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {

}

func ShowAccountInfo(chatID int64, userID int64) tgbotapi.MessageConfig {

	balResp, err := http.Get("https://phunkao.fun:8443/vb-api?uid=" + fmt.Sprint(userID))
	if err != nil {
		fmt.Println("Error fetching balance:", err)
	}
	defer balResp.Body.Close()

	if balResp.StatusCode != http.StatusOK {
		fmt.Println("Non-OK HTTP status:", balResp.StatusCode)
	}

	var balance int64 = 0
	err = json.NewDecoder(balResp.Body).Decode(&balance)
	if err != nil {
		fmt.Println("Error decoding balance response:", err)
	}

	msg := tgbotapi.NewMessage(chatID, "Ваш профиль:\nID пользователя: "+fmt.Sprint(userID)+"\nБаланс: "+fmt.Sprint(userID)+" рублей")
	return msg
}
