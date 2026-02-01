package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"verbose-bassoon/bot/account"
	"verbose-bassoon/bot/shop"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var bot *tgbotapi.BotAPI
var topupers = make(map[int64]string)

func main() {

	token := os.Getenv("TG_BOT_TOKEN")
	if token == "" {
		log.Fatal("TG_BOT_TOKEN environment variable not set")
	}

	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Fatal("Bot create FAIL:", err)
	}

	bot.Debug = true
	log.Printf("Auth as: @%s", bot.Self.UserName)

	webhookURL := "https://www.phunkao.fun:8443/vb-wh"
	webhook, _ := tgbotapi.NewWebhook(webhookURL)

	webhook.AllowedUpdates = []string{"message", "callback_query"}

	_, err = bot.Request(webhook)
	if err != nil {
		log.Fatal("Setting webhook FAIL:", err)
	}
	log.Println("Webhook setted:", webhookURL)

	updates := bot.ListenForWebhook("/vb-wh")

	go func() {
		http.HandleFunc("/vb-notify", notifyHandler)

		if err := http.ListenAndServe(":8011", nil); err != nil {
			log.Fatal("HTTP WebHook-Server FAULT:", err)
		}
		log.Println("Go back listening :8011 (HTTP)")
	}()

	for update := range updates {
		log.Printf("Get update: %+v", update)

		if topupers[update.Message.Chat.ID] == "cb" {
			paymentSum := strings.TrimSpace(update.Message.Text)
			amount, err := strconv.ParseFloat(paymentSum, 64)

			if err != nil || amount < 50 {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Ошибка: введите корректную сумму (число не менее 50).")
				bot.Send(msg)
				continue
			}

			msg := account.CreateCryptoInvoice(update.Message.Chat.ID, update.Message.From.ID, float64(amount))
			bot.Send(msg)
			return
		}

		if update.Message != nil && update.Message.IsCommand() {
			if update.Message.Command() == "start" {

				account.Init(update.Message.Chat.ID, update.Message.From.ID)

				keyboard := tgbotapi.NewReplyKeyboard(
					tgbotapi.NewKeyboardButtonRow(
						tgbotapi.NewKeyboardButton("🔌 Магазин"),
						tgbotapi.NewKeyboardButton("👤 Профиль"),
					),
					tgbotapi.NewKeyboardButtonRow(
						tgbotapi.NewKeyboardButton("🧩 Тех.Поддержка"),
						tgbotapi.NewKeyboardButton("🕸 Личный ВПН"),
					),
				)
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Привет! Ты попал в автоматизированный магазин цифровых товаров!")
				msg.ReplyMarkup = keyboard
				bot.Send(msg)
			}
		}

		if update.Message != nil {
			switch update.Message.Text {
			case "🔌 Магазин":
				msg := shop.ShowShopMenu(update.Message.Chat.ID)
				bot.Send(msg)
			case "👤 Профиль":
				msg := account.ShowAccountInfo(update.Message.Chat.ID, update.Message.From.ID)
				bot.Send(msg)
			case "🧩 Тех.Поддержка":
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Свяжитесь с нашей тех. поддержкой")
				bot.Send(msg)
			case "🕸 Личный ВПН":
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Ваш личный VPN менеджер!")
				keyboard := tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonURL("Перейти в VPN Менеджер", "https://t.me/surfboost_bot?start=ref287657335"),
					),
				)
				msg.ReplyMarkup = keyboard
				bot.Send(msg)
			}
		}

		if update.CallbackQuery != nil {

			cbDataParts := strings.Split(update.CallbackQuery.Data, ":")

			if len(cbDataParts) == 1 {
				switch cbDataParts[0] {
				case "payments":

					msg := account.ShowPaymentMenu(update.CallbackQuery.Message.Chat.ID)

					editMsg := tgbotapi.NewEditMessageTextAndMarkup(
						update.CallbackQuery.Message.Chat.ID,
						update.CallbackQuery.Message.MessageID,
						msg.Text,
						msg.ReplyMarkup.(tgbotapi.InlineKeyboardMarkup),
					)
					bot.Send(editMsg)
				case "help":
					msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "Свяжитесь с нашей тех. поддержкой")
					bot.Send(msg)
				}
			}

			if len(cbDataParts) == 2 {
				switch cbDataParts[0] + ":" + cbDataParts[1] {
				case "payments:cb":
					topupers[update.CallbackQuery.Message.Chat.ID] = "cb"
					input_sum_msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "Введите сумму для пополнения баланса в рублях (мин. 50 руб.):")
					bot.Send(input_sum_msg)
				}
			}

			callback := tgbotapi.NewCallback(update.CallbackQuery.ID, "")
			bot.Request(callback)
		}
	}
}

func notifyHandler(w http.ResponseWriter, r *http.Request) {
	type internalSendReq struct {
		Cid  string `json:"cid"`
		Text string `json:"text"`
	}
	var req internalSendReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fmt.Printf("BAD notify JSON")
		return
	}
	if req.Cid == "" || strings.TrimSpace(req.Text) == "" {
		fmt.Printf("missing cid/text")
		return
	}
	cid, _ := strconv.ParseInt(req.Cid, 10, 64)
	msg := tgbotapi.NewMessage(cid, req.Text)
	if _, err := bot.Send(msg); err != nil {
		log.Println("send fail:", err)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}
