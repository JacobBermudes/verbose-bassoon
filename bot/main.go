package main

import (
	"log"
	"net/http"
	"os"

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

	keySender := int64(0)

	for update := range updates {
		log.Printf("Get update: %+v", update)

		if update.Message != nil && update.Message.IsCommand() {
			if update.Message.Command() == "start" {
				keyboard := tgbotapi.NewReplyKeyboard(
					tgbotapi.NewKeyboardButtonRow(
						tgbotapi.NewKeyboardButton("Магазин"),
						tgbotapi.NewKeyboardButton("Профиль"),
					),
					tgbotapi.NewKeyboardButtonRow(
						tgbotapi.NewKeyboardButton("Тех.Поддержка"),
						tgbotapi.NewKeyboardButton("Личный ВПН"),
					),
				)
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Привет! Ты попал в автоматизированный магазин цифровых товаров!")
				msg.ReplyMarkup = keyboard
				bot.Send(msg)
			}
		}

		if update.Message != nil && keySender == update.Message.From.ID {

		}

		if update.CallbackQuery != nil {
			callback := tgbotapi.NewCallback(update.CallbackQuery.ID, "")
			bot.Request(callback)
			msg := tgbotapi.NewMessage(update.FromChat().ID, update.CallbackQuery.Data)
			bot.Send(msg)
		}
	}
}
