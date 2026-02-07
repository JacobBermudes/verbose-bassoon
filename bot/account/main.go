package account

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var apiAddres = os.Getenv("API_ADDRESS")
var apiPort = os.Getenv("API_PORT")

func main() {

}

func Init(cid int64, uid int64) {

	if apiAddres == "" || apiPort == "" {
		fmt.Printf("API_PORT or API_ADDRESS environment variable not setted!\n")
	}

	var accInitReq struct {
		Amount   float64 `json:"amount"`
		Uid      int64   `json:"uid"`
		VbMethod string  `json:"vbMethod"`
		Data     string  `json:"data"`
	}

	accInitReq.Amount = 0
	accInitReq.Uid = uid
	accInitReq.VbMethod = "accountInit"
	accInitReq.Data = strconv.FormatInt(cid, 10)

	payload, err := json.Marshal(accInitReq)
	if err != nil {
		log.Println("Error encoding JSON:", err)
	}

	http.Post(apiAddres+":"+apiPort+"/vb-api/v1", "application/json", bytes.NewBuffer(payload))
}

func ShowAccountInfo(chatID int64, userID int64, username string) tgbotapi.MessageConfig {

	if apiAddres == "" || apiPort == "" {
		fmt.Printf("API_PORT or API_ADDRESS environment variable not setted!\n")
	}

	var getBalance struct {
		Amount   float64 `json:"amount"`
		Uid      int64   `json:"uid"`
		VbMethod string  `json:"vbMethod"`
		Data     string  `json:"data"`
	}
	getBalance.Uid = userID
	getBalance.VbMethod = "getBalance"

	payload, err := json.Marshal(getBalance)
	if err != nil {
		fmt.Println("Get balance request JSON marshal error")
	}
	balResp, err := http.Post(apiAddres+":"+apiPort+"/vb-api/v1", "application/json", bytes.NewBuffer(payload))
	if err != nil {
		fmt.Printf("Fail to req balance from api!")
	}
	if balResp.StatusCode != http.StatusOK {
		fmt.Println("Non-OK HTTP status:", balResp.StatusCode)
	}

	var balance int64 = 0
	err = json.NewDecoder(balResp.Body).Decode(&balance)
	if err != nil {
		fmt.Println("Error decoding balance response:", err)
	}

	msg := tgbotapi.NewMessage(chatID, "\n 👨 Ваш профиль: "+username+" (ID "+fmt.Sprint(userID)+")\n\n 💰 Баланс: "+fmt.Sprint(balance)+" рублей")

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
			tgbotapi.NewInlineKeyboardButtonData("⚖️ Юридическая информация", "license"),
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
			tgbotapi.NewInlineKeyboardButtonData("Platega(Рубли, СБП)", "payments:pltg"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Crypto BOT (TG)", "payments:cb"),
			tgbotapi.NewInlineKeyboardButtonData("Crypto", "payments:crypto"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Главное меню", "mainmenu"),
		),
	)
	msg.ReplyMarkup = keyboard

	return msg
}

func CreateCryptoBotInvoice(chatID int64, userID int64, amount float64) tgbotapi.MessageConfig {

	var createInvoiceResp struct {
		Amount   float64 `json:"amount"`
		Uid      int64   `json:"uid"`
		VbMethod string  `json:"vbMethod"`
		Data     string  `json:"data"`
	}

	createInvoiceResp.Amount = amount
	createInvoiceResp.Uid = userID
	createInvoiceResp.VbMethod = "createCryptoInvoice"

	payloadBytes, err := json.Marshal(createInvoiceResp)
	if err != nil {
		log.Println("Error encoding JSON:", err)
	}
	internalResp, err := http.Post(apiAddres+":"+apiPort+"/vb-api/v1", "application/json", bytes.NewBuffer(payloadBytes))
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

	msg := tgbotapi.NewMessage(chatID, "Кнопка для оплаты:")
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

func CreateCryptoExchange(chatID int64, userID int64, amount float64, coin string) tgbotapi.MessageConfig {
	var createInvoiceResp struct {
		Amount   float64 `json:"amount"`
		Uid      int64   `json:"uid"`
		VbMethod string  `json:"vbMethod"`
		Data     string  `json:"data"`
	}

	createInvoiceResp.Amount = amount
	createInvoiceResp.Uid = userID
	createInvoiceResp.VbMethod = "createCryptoExchange"
	createInvoiceResp.Data = coin

	payloadBytes, err := json.Marshal(createInvoiceResp)
	if err != nil {
		log.Println("Error encoding JSON:", err)
	}
	internalResp, err := http.Post(apiAddres+":"+apiPort+"/vb-api/v1", "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		log.Println("Error creating invoice:", err)
	}
	defer internalResp.Body.Close()

	respBody, err := io.ReadAll(internalResp.Body)
	if err != nil {
		log.Println("Error reading response body:", err)
	}

	msg := tgbotapi.NewMessage(chatID, "Ожидаем на оплату "+string(respBody)+" "+coin)
	msg.ParseMode = "Markdown"

	return msg
}
