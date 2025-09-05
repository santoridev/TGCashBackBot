package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/joho/godotenv"
)

//sub

const adminID int64 = 1 /// change to your admin id

type UserForm struct {
	DepositRange  string
	DepositAmount int
	ReturnType    string // "card" или "crypto"
	Cashback      int
	Bank          string
	CardNumber    string
	CryptoNetwork string
	CryptoAddress string
	ID            string
	Photos        []string
	Step          int
}

var userForms = make(map[int64]*UserForm)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	dispatcher := func(ctx context.Context, b *bot.Bot, update *models.Update) {
		if update.Message != nil {
			if update.Message.Text == "/start" {
				startHandler(ctx, b, update)
			} else {
				messageHandler(ctx, b, update)
			}
		} else if update.CallbackQuery != nil {
			callbackHandler(ctx, b, update)
		} else {
			defaultHandler(ctx, b, update)
		}
	}

	opts := []bot.Option{
		bot.WithDefaultHandler(dispatcher),
	}
	ERR := godotenv.Load()
	if ERR != nil {
		log.Fatal("Error loading .env file")
	}
	b, err := bot.New(os.Getenv("TELEGRAM"), opts...)
	if err != nil {
		panic(err)
	}
	b.Start(ctx)
}

func startHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	kb := &models.ReplyKeyboardMarkup{
		Keyboard: [][]models.KeyboardButton{
			{{Text: "от 500 до 1001 ₽"}},
			{{Text: "от 1001 до 1499 ₽"}},
			{{Text: "1500+ ₽"}},
		},
		ResizeKeyboard:  true,
		OneTimeKeyboard: true,
	}

	text := "👋Привет " + update.Message.From.FirstName + " !\n\n" +
		"Это бот для кэшбека моим рефам. Следуй дальнейшим шагам, чтобы завершить проверку." +
		"\n\n🎁 анонсы + розыгрыши --> @mlaffon" +
		"\n\nВыберите, какой депозит вы сделали:"

	mediaGroup := []models.InputMedia{
		&models.InputMediaPhoto{
			Media:   "https://i.ibb.co/N2VP8zjx/promo.webp",
			Caption: text,
		},
	}

	_, err := b.SendMediaGroup(ctx, &bot.SendMediaGroupParams{
		ChatID: update.Message.Chat.ID,
		Media:  mediaGroup,
	})
	if err != nil {
		log.Printf("Error sending media group: %v", err)
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      update.Message.Chat.ID,
		Text:        "👇 из меню ниже:",
		ReplyMarkup: kb,
	})

	userForms[update.Message.Chat.ID] = &UserForm{Step: 1}
}

func sendReturnTypeButtons(ctx context.Context, b *bot.Bot, chatID int64) {
	kb := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "Карта 💳", CallbackData: "return_card"},
				{Text: "Крипта ₿", CallbackData: "return_crypto"},
			},
		},
	}
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        "Куда хотите возврат?",
		ReplyMarkup: kb,
	})
}

func callbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}
	chatID := update.CallbackQuery.From.ID

	if strings.HasPrefix(update.CallbackQuery.Data, "reply_") {
		userIDStr := strings.TrimPrefix(update.CallbackQuery.Data, "reply_")
		userID, _ := strconv.ParseInt(userIDStr, 10, 64)
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: userID,
			Text:   "Админ хочет с вами связаться по вашей заявке.",
		})
		b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
		})
		return
	}

	if strings.HasPrefix(update.CallbackQuery.Data, "paid_") {
		userIDStr := strings.TrimPrefix(update.CallbackQuery.Data, "paid_")
		userID, _ := strconv.ParseInt(userIDStr, 10, 64)

		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: userID,
			Text:   "✅ Ваша заявка оплачена! 💸 Деньги скоро придут.",
		})

		b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			Text:            "Заявка отмечена как оплаченная ✅",
			ShowAlert:       false,
		})
		return
	}

	form, ok := userForms[chatID]
	if !ok {
		return
	}

	switch update.CallbackQuery.Data {
	case "return_card":
		form.ReturnType = "card"
		form.Step = 21
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "Введите номер карты (16 цифр):",
		})
	case "return_crypto":
		form.ReturnType = "crypto"
		form.Step = 22
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "Выберите сеть для USDT (например: TRC20, ERC20, BSC):",
		})

	case "confirm_data":
		form.Step = 4

		mediaGroup := []models.InputMedia{
			&models.InputMediaPhoto{
				Media: "https://i.ibb.co/JwNNpKVR/trans.webp",
				Caption: `Пришлите 2 скрина:
	1. Профиль — где видно ваш ID и баланс.
	2. История транзакций — вкладка "Депозиты", где отображается пополнение.
	
	Скрины нужны для подтверждения депозита по моей рефке.`,
			},
			&models.InputMediaPhoto{
				Media: "https://i.ibb.co/s9t2MBK8/profile.webp",
			},
		}

		_, err := b.SendMediaGroup(ctx, &bot.SendMediaGroupParams{
			ChatID: chatID,
			Media:  mediaGroup,
		})
		if err != nil {
			log.Printf("Error sending media group: %v", err)
		}

	case "edit_data":
		form.Step = 1
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "Хорошо, давайте начнём заново.\nВыберите, какой депозит вы закинули:",
			ReplyMarkup: &models.ReplyKeyboardMarkup{
				Keyboard: [][]models.KeyboardButton{
					// {{Text: "100 - 700 ₽"}},
					{{Text: "от 500 до 1000 ₽"}},
					{{Text: "от 1001 до 1499 ₽"}},
					// {{Text: "1500 - 2000 ₽"}},
					{{Text: "1500+ ₽"}},
				},
				ResizeKeyboard:  true,
				OneTimeKeyboard: true,
			},
		})
	}

	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
	})
}

func messageHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	chatID := update.Message.Chat.ID
	form, ok := userForms[chatID]
	if !ok {
		return
	}

	switch form.Step {
	case 1:
		form.DepositRange = update.Message.Text
		form.Step = 11

		var percent string
		switch form.DepositRange {
		case "от 500 до 1000 ₽":
			percent = "70%"
		case "от 1000 до 1499 ₽":
			percent = "50%"
		case "1500+ ₽":
			percent = "30% (⚠ максимум 1500₽)"
		default:
			percent = "неизвестно"
		}

		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text: "Вы выбрали: " + form.DepositRange +
				"\nКэшбэк для этого диапазона: " + percent +
				"\n\nВведите точную сумму вашего депозита:",
		})

	case 11:
		amount, err := strconv.Atoi(update.Message.Text)
		if err != nil || amount <= 0 {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: chatID,
				Text:   "Введите корректную сумму (число).",
			})
			return
		}

		valid := false
		switch form.DepositRange {
		case "от 500 до 1000 ₽":
			if amount >= 500 && amount <= 1000 {
				valid = true
				form.Cashback = amount * 70 / 100
			}
		case "от 1001 до 1499 ₽":
			if amount >= 1001 && amount <= 1499 {
				valid = true
				form.Cashback = amount * 50 / 100
			}
		case "1500+ ₽":
			if amount >= 1500 {
				valid = true
				form.Cashback = amount * 30 / 100
				if form.Cashback > 1500 {
					form.Cashback = 1500
				}
			}
		}

		if !valid {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: chatID,
				Text:   "Сумма не входит в выбранный диапазон. Введите корректное значение:",
			})
			return
		}

		form.DepositAmount = amount
		form.Step = 2
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "✅ Сумма принята! Кэшбэк составит: " + strconv.Itoa(form.Cashback) + " ₽.\n\nТеперь выберите, куда хотите возврат:",
		})
		sendReturnTypeButtons(ctx, b, chatID)

	case 21:
		cardNumber := update.Message.Text
		matched, _ := regexp.MatchString(`^\d{16}$`, cardNumber)
		if !matched {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: chatID,
				Text:   "Неверный номер карты. Введите 16 цифр:",
			})
			return
		}
		form.CardNumber = cardNumber
		form.Step = 211
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "Введите название банка:",
		})
	case 211:
		form.Bank = update.Message.Text
		form.Step = 3
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "Введите ваше ID:",
		})

	// Крипта
	case 22:
		form.CryptoNetwork = update.Message.Text
		form.Step = 221
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "Введите адрес кошелька:",
		})
	case 221:
		form.CryptoAddress = update.Message.Text
		form.Step = 3
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "Введите ваше ID:",
		})

		// ID пользователя
	case 3:
		id := update.Message.Text
		matched, _ := regexp.MatchString(`^\d+$`, id)
		if !matched {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: chatID,
				Text:   "❌ ID должен содержать только цифры. Введите корректный ID:",
			})
			return
		}

		form.ID = id

		preview := "Проверьте введённые данные:\n\n" +
			"💰 Депозит: " + strconv.Itoa(form.DepositAmount) + " ₽ (" + form.DepositRange + ")\n" +
			"🎁 Кэшбэк: " + strconv.Itoa(form.Cashback) + " ₽\n" +
			"💳 Тип возврата: "
		if form.ReturnType == "card" {
			preview += "Карта\n" +
				"🏦 Банк: " + form.Bank + "\n" +
				"🔢 Номер карты: " + form.CardNumber + "\n"
		} else {
			preview += "Крипта\n" +
				"🌐 Сеть: " + form.CryptoNetwork + "\n" +
				"💼 Адрес: " + form.CryptoAddress + "\n"
		}
		preview += "👤 ID: " + form.ID + "\n\n" +
			"Если всё верно — нажмите «Всё верно».\n" +
			"Если нужно исправить — нажмите «Изменить»."

		// Клавиатура с выбором
		kb := &models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{
					{Text: "✅ Всё верно", CallbackData: "confirm_data"},
					{Text: "✏ Изменить", CallbackData: "edit_data"},
				},
			},
		}

		// Отправляем превью с кнопками
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:      chatID,
			Text:        preview,
			ReplyMarkup: kb,
		})

	// Фото
	case 4:
		if len(update.Message.Photo) > 0 {
			fileID := update.Message.Photo[len(update.Message.Photo)-1].FileID
			form.Photos = append(form.Photos, fileID)
			if len(form.Photos) == 1 {
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: chatID,
					Text:   "Фото получено! Пришлите ещё или напишите /done, когда закончите.",
				})
			}
		} else if update.Message.Text == "/done" {
			if len(form.Photos) == 0 {
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: chatID,
					Text:   "Вы не отправили ни одного фото.",
				})
				return
			}
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: chatID,
				Text:   "Спасибо! Ваша заявка отправлена на проверку.",
			})
			sendFormToAdmin(ctx, b, form, chatID, update.Message.From.Username)
			delete(userForms, chatID)
		} else {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: chatID,
				Text:   "Пожалуйста, отправьте скрины как фото, а не как файл.",
			})
		}
	}
}

// Отправка заявки админу с кнопкой "Ответить пользователю"
func sendFormToAdmin(ctx context.Context, b *bot.Bot, form *UserForm, userID int64, username string) {
	// Формируем текст заявки
	text := "Новая заявка:\n" +
		"Депозит: " + form.DepositRange + "\n" +
		"Возврат: " + form.ReturnType + "\n" +
		"Кэшбэк: " + strconv.Itoa(form.Cashback) + " ₽\n"

	if form.ReturnType == "card" {
		text += "Банк: " + form.Bank + "\n" +
			"Номер карты: " + form.CardNumber + "\n"
	} else if form.ReturnType == "crypto" {
		text += "Сеть: " + form.CryptoNetwork + "\n" +
			"Адрес: " + form.CryptoAddress + "\n"
	}

	text += "ID: " + form.ID

	kb := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{
					Text: "Перейти в чат с пользователем",
					URL:  "https://t.me/" + username,
				},
			},
			{
				{
					Text:         "Оплатил ✅",
					CallbackData: "paid_" + strconv.FormatInt(userID, 10),
				},
			},
		},
	}

	if len(form.Photos) > 0 {

		media := make([]models.InputMedia, 0, len(form.Photos))
		for i, fileID := range form.Photos {
			if i == 0 {
				media = append(media, &models.InputMediaPhoto{
					Media:   fileID,
					Caption: text,
				})
			} else {
				media = append(media, &models.InputMediaPhoto{
					Media: fileID,
				})
			}
		}

		b.SendMediaGroup(ctx, &bot.SendMediaGroupParams{
			ChatID: adminID,
			Media:  media,
		})

		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:      adminID,
			Text:        "⬆️ Заявка выше",
			ReplyMarkup: kb,
		})
	} else {

		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:      adminID,
			Text:        text,
			ReplyMarkup: kb,
		})
	}
}

func defaultHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Пожалуйста, используйте клавиатуру для выбора опций.",
		})
	}
}
