package shop

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

func main() {

}

func ShowShopMenu(chatID int64) tgbotapi.MessageConfig {

	msg := tgbotapi.NewMessage(chatID, "Добро пожаловать в магазин! Выберите категорию товара:")
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Stars", "category_stars"),
			tgbotapi.NewInlineKeyboardButtonData("Accounts", "category_accounts"),
		),
	)
	msg.ReplyMarkup = keyboard
	return msg
}
