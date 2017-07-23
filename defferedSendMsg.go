package telebot

import (
	"time"
	"reflect"
)

// источник https://habrahabr.ru/post/317666/

// Мап каналов для сообщений, где ключом является id пользователя
var deferredMessages = make(map[string]chan DeferredMessage)
// Здесь будем хранить время последней отправки сообщения для каждого пользователя
var lastMessageTimes = make(map[string]int64)

// callback будем вызывать для обработки ошибок при обращении к API
type DeferredMessage struct {
	Recipient Recipient
	MsgType   string
	Message   string
	Photo     *Photo
	Sticker   *Sticker
	Doc       *Document
	Action    string
	Options   *SendOptions
	Callback  func(*MsgResult, error)
}

// Метод для отправки отложенного сообщения
func SendMsgDeferred(dm *DeferredMessage) {

	chatId := dm.Recipient.Destination()
	if _, ok := deferredMessages[chatId]; !ok {
		deferredMessages[chatId] = make(chan DeferredMessage, 1000)
	}

	deferredMessages[chatId] <- *dm
}

func (b *Bot) SendDeferredMessages(msgInSec int) {
	// Создаем тикер с заданной периодичностью сообщений в секунду
	if msgInSec == 0 {
		msgInSec = 30 // дефолтная периодичность 1/30 секунд
	}
	timer := time.NewTicker(time.Second/ time.Duration(msgInSec))

	for range timer.C {
		// Формируем массив SelectCase'ов из каналов, пользователи которых готовы получить следующее сообщение
		cases := []reflect.SelectCase{}
		for userId, ch := range deferredMessages {
			if userCanReceiveMessage(userId) && len(ch) > 0 {
				// Формирование case
				cs := reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(ch)}
				cases = append(cases, cs)
			}
		}

		if len(cases) > 0 {
			// Достаем одно сообщение из всех каналов
			_, value, ok := reflect.Select(cases)

			if ok {
				dm := value.Interface().(DeferredMessage)
				// Выполняем запрос к API

				result, err := sendMsg(b, dm)
				if dm.Callback != nil {
					dm.Callback(result, err)
				}
				// Записываем пользователю время последней отправки сообщения.
				lastMessageTimes[dm.Recipient.Destination()] = time.Now().UnixNano()
			}
		}
	}
}

// Проверка может ли уже пользователь получить следующее сообщение
func userCanReceiveMessage(userId string) bool {
	t, ok := lastMessageTimes[userId]

	return !ok || t + int64(time.Second/2) <= time.Now().UnixNano()
}

func sendMsg(b *Bot, dm DeferredMessage) (result *MsgResult, err error) {
	switch dm.MsgType {
	case "photo":
		err = b.SendPhoto(dm.Recipient, dm.Photo, dm.Options)
	case "sticker":
		err = b.SendSticker(dm.Recipient, dm.Sticker, dm.Options)
	case "doc":
		err = b.SendDocument(dm.Recipient, dm.Doc, dm.Options)
	case "text":
		result, err = b.SendMessage(dm.Recipient, dm.Message, dm.Options)
	case "action":
		err = b.SendChatAction(dm.Recipient, dm.Action)
	default:
		result, err = b.SendMessage(dm.Recipient, dm.Message, nil)
	}
	return
}