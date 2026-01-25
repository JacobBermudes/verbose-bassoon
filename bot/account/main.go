package account

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
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

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💲 Пополнение баланса", "payments"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💸 История заказов", "donate"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💵 Акция «Приведи друга»", "referral"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💬 Помощь", "help"),
		),
	)
	msg.ReplyMarkup = keyboard

	return msg
}

func ShowPaymentMenu(chatID int64) tgbotapi.MessageConfig {
	msg := tgbotapi.NewMessage(chatID, "Выберите сумму для пополнения баланса:")

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Crypto BOT (telegram)", "payments:cb"),
			tgbotapi.NewInlineKeyboardButtonData("СБП QR", "payments:sbp"),
		),
	)
	msg.ReplyMarkup = keyboard

	return msg
}

func CreateCryptoInvoice(chatID int64, userID int64, amount float64) tgbotapi.MessageConfig {

	var createInvoiceResp struct {
		Amount   float64 `json:"amount"`
		Uid      int64   `json:"uid"`
		VbMethod string  `json:"vbMethod"`
	}

	createInvoiceResp.Amount = amount
	createInvoiceResp.Uid = userID
	createInvoiceResp.VbMethod = "createCryptoInvoice"

	// Call API to create invoice
	payloadBytes, err := json.Marshal(createInvoiceResp)
	if err != nil {
		log.Println("Error encoding JSON:", err)
	}
	internalResp, err := http.Post("https://www.phunkao.fun:8443/vb-api/v1", "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		log.Println("Error creating invoice:", err)
	}
	defer internalResp.Body.Close()

	respBody, err := io.ReadAll(internalResp.Body)
	if err != nil {
		log.Println("Error reading response body:", err)
	}

	var responseData map[string]interface{}
	err = json.Unmarshal(respBody, &responseData)
	if err != nil {
		log.Println("Error decoding invoice response:", err)
	}

	payURL := ""
	if url, ok := responseData["pay_url"].(string); ok {
		payURL = url
	} else if result, ok := responseData["result"].(map[string]interface{}); ok {
		if url, ok := result["pay_url"].(string); ok {
			payURL = url
		}
	}
	if payURL == "" {
		log.Println("Warning: pay_url is empty")
	}

	stringAmount := fmt.Sprintf("%.2f", amount)
	msg := tgbotapi.NewMessage(chatID, "Вы хотите пополнить баланс на сумму: "+stringAmount+" рублей.\n\nПерейдите по ссылке для оплаты:")
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{
			{
				tgbotapi.NewInlineKeyboardButtonURL("Оплатить "+fmt.Sprint(amount)+" руб.", payURL),
			},
		},
	}

	return msg
}
