package main

import (
	"bytes"
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
		log.Println("Go back listening :8011 (HTTP)")

		if err := http.ListenAndServe(":8011", nil); err != nil {
			log.Fatal("HTTP WebHook-Server FAULT:", err)
		}
	}()

	for update := range updates {
		log.Printf("Get update: %+v", update)

		if update.Message != nil && update.Message.IsCommand() {
			if update.Message.Command() == "start" {
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

		if update.Message != nil && update.Message.ReplyToMessage != nil {
			if update.Message.ReplyToMessage.Text == "Введите сумму для пополнения баланса в рублях (мин. 50 руб.):" {
				paymentSum := strings.TrimSpace(update.Message.Text)

				var createInvoiceResp struct {
					Amount   int64  `json:"amount"`
					Uid      string `json:"uid"`
					VbMethod string `json:"vbMethod"`
				}
				createInvoiceResp.Amount, _ = strconv.ParseInt(paymentSum, 10, 64)
				createInvoiceResp.Uid = fmt.Sprint(update.Message.From.ID)
				createInvoiceResp.VbMethod = "createInvoice"

				// Call API to create invoice
				payloadBytes, err := json.Marshal(createInvoiceResp)
				if err != nil {
					log.Println("Error encoding JSON:", err)
					continue
				}

				internalResp, err := http.Post("https://www.phunkao.fun:8443/vb-api/v1", "application/json", bytes.NewBuffer(payloadBytes))
				var invoiceLink struct {
					InvoiceURL string `json:"pay_url"`
				}
				if err != nil {
					log.Println("Error creating invoice:", err)
				} else {
					defer internalResp.Body.Close()
					err = json.NewDecoder(internalResp.Body).Decode(&invoiceLink)
					if err != nil {
						log.Println("Error decoding invoice response:", err)
					}
				}

				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Вы хотите пополнить баланс на сумму: "+paymentSum+" рублей.\n\nПерейдите по ссылке для оплаты:")
				msg.ParseMode = "Markdown"
				msg.Text += "\n[Оплатить " + paymentSum + " руб.](" + invoiceLink.InvoiceURL + ")"
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
					input_sum_msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "Введите сумму для пополнения баланса в рублях (мин. 100 руб.):")
					input_sum_msg.ReplyMarkup = tgbotapi.ForceReply{ForceReply: true, Selective: true}
					bot.Send(input_sum_msg)
				}
			}

			callback := tgbotapi.NewCallback(update.CallbackQuery.ID, "")
			bot.Request(callback)
		}
	}
}
