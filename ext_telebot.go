// ADDED BY DROP - https://github.com/matryer/drop (v0.7)
//  source: github.com/tucnak/telebot (41796c460e2f38cfd32062dd27eed6d4ee40d7ba)
//  update: drop -f github.com/tucnak/telebot
// license: The MIT License (MIT) (see repo for details)

// Package telebot provides a handy wrapper for interactions
// with Telegram bots.
//
// Here is an example of helloworld bot implementation:
//
//	import (
//		"time"
//		"github.com/tucnak/telebot"
//	)
//
//	func main() {
//		bot, err := telebot.NewBot("SECRET_TOKEN")
//		if err != nil {
//			return
//		}
//
//		messages := make(chan telebot.Message)
//		bot.Listen(messages, 1*time.Second)
//
//		for message := range messages {
//			if message.Text == "/hi" {
//				bot.SendMessage(message.Chat,
//					"Hello, "+message.Sender.FirstName+"!", nil)
//			}
//		}
//	}
//
package telebot

// A bunch of available chat actions.
const (
	Typing            = "typing"
	UploadingPhoto    = "upload_photo"
	UploadingVideo    = "upload_video"
	UploadingAudio    = "upload_audio"
	UploadingDocument = "upload_document"
	RecordingVideo    = "record_video"
	RecordingAudio    = "record_audio"
	FindingLocation   = "find_location"
)
