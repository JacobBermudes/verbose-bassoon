package account

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {

}

func ShowAccountInfo(chatID int64, userID int64) tgbotapi.MessageConfig {

	apiAddres := os.Getenv("API_ADDRESS")
	apiPort := os.Getenv("API_PORT")
	if apiAddres == "" || apiPort == "" {
		fmt.Printf("API_PORT or API_ADDRESS environment variable not setted!\n")
	}

	balResp, err := http.Get(apiAddres + ":" + apiPort + "/vb-api/balance?uid=" + fmt.Sprint(userID))
	if err != nil && apiAddres != "" && apiPort != "" {
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

	msg := tgbotapi.NewMessage(chatID, "Ваш профиль:\nID пользователя: "+fmt.Sprint(userID)+"\nБаланс: "+fmt.Sprint(balance)+" рублей")
	return msg
}
