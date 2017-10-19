// ADDED BY DROP - https://github.com/matryer/drop (v0.7)
//  source: github.com/tucnak/telebot (41796c460e2f38cfd32062dd27eed6d4ee40d7ba)
//  update: drop -f github.com/tucnak/telebot
// license: The MIT License (MIT) (see repo for details)

package telebot

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"
	"github.com/pkg/errors"
)

// Bot represents a separate Telegram bot instance.
type Bot struct {
	Token     string
	Identity  User
	Messages  chan Message
	Queries   chan Query
	Callbacks chan Callback
}

// NewBot does try to build a Bot with token `token`, which
// is a secret API key assigned to particular bot.
func NewBot(token string) (*Bot, error) {
	user, err := getMe(token)
	if err != nil {
		return nil, err
	}

	return &Bot{
		Token:    token,
		Identity: user,
	}, nil
}

// Listen periodically looks for updates and delivers new messages
// to the subscription channel.
func (b *Bot) Listen(subscription chan Message, timeout time.Duration) {
	go b.poll(subscription, nil, nil, timeout)
}

// Start periodically polls messages and/or updates to corresponding channels
// from the bot object.
func (b *Bot) Start(timeout time.Duration) {
	b.poll(b.Messages, b.Queries, b.Callbacks, timeout)
}

func (b *Bot) poll(
messages chan Message,
queries chan Query,
callbacks chan Callback,
timeout time.Duration,
) {
	latestUpdate := 0

	for {
		updates, err := getUpdates(b.Token,
			latestUpdate + 1,
			int(timeout / time.Second),
		)

		if err != nil {
			log.Println("failed to get updates:", err)
			continue
		}

		for _, update := range updates {
			if update.Payload != nil /* if message */ {
				if messages == nil {
					continue
				}

				messages <- *update.Payload
			} else if update.Query != nil /* if query */ {
				if queries == nil {
					continue
				}

				queries <- *update.Query
			} else if update.Callback != nil {
				if callbacks == nil {
					continue
				}

				callbacks <- *update.Callback
			}

			latestUpdate = update.ID
		}
	}

}

type MsgResult struct {
	Message_id int
}

type BotInfo struct {
	Id int64 `json:"id"`
	FirstName string `json:"first_name"`
	Username string `json:"username"`
}

// SendMessage sends a text message to recipient.
func (b *Bot) SendMessage(recipient Recipient, message string, options *SendOptions) (result *MsgResult, Error error) {
	params := map[string]string{
		"chat_id": recipient.Destination(),
		"text":    message,
	}

	if options != nil {
		embedSendOptions(params, options)
	}

	responseJSON, err := sendCommand("sendMessage", b.Token, params)
	if err != nil {
		return nil, err
	}

	var responseRecieved struct {
		Ok          bool
		Description string
		Result      MsgResult
	}

	err = json.Unmarshal(responseJSON, &responseRecieved)
	if err != nil {
		return nil, err
	}

	if !responseRecieved.Ok {
		return nil, fmt.Errorf("telebot: %s", responseRecieved.Description)
	}

	return &responseRecieved.Result, nil
}

// ForwardMessage forwards a message to recipient.
func (b *Bot) ForwardMessage(recipient Recipient, message Message) error {
	params := map[string]string{
		"chat_id":      recipient.Destination(),
		"from_chat_id": strconv.Itoa(message.Origin().ID),
		"message_id":   strconv.Itoa(message.ID),
	}

	responseJSON, err := sendCommand("forwardMessage", b.Token, params)
	if err != nil {
		return err
	}

	var responseRecieved struct {
		Ok          bool
		Description string
	}

	err = json.Unmarshal(responseJSON, &responseRecieved)
	if err != nil {
		return err
	}

	if !responseRecieved.Ok {
		return fmt.Errorf("telebot: %s", responseRecieved.Description)
	}

	return nil
}


// EditMessage sends a text message to recipient.
func (b *Bot) EditMessageText(message Message, text string, options *SendOptions) error {
	params := map[string]string{
		"chat_id":    strconv.FormatInt(message.Chat.ID, 10),
		"message_id": strconv.Itoa(message.ID),
		"text":       text,
	}

	if options != nil {
		embedSendOptions(params, options)
	}

	responseJSON, err := sendCommand("editMessageText", b.Token, params)
	if err != nil {
		return err
	}

	var responseRecieved struct {
		Ok          bool
		Description string
	}

	err = json.Unmarshal(responseJSON, &responseRecieved)
	if err != nil {
		return err
	}

	if !responseRecieved.Ok {
		return fmt.Errorf("telebot: %s", responseRecieved.Description)
	}

	return nil
}

func (b *Bot) DeleteMessage(message Message) error {
	params := map[string]string{
		"chat_id":    strconv.FormatInt(message.Chat.ID, 10),
		"message_id": strconv.Itoa(message.ID),
	}
	responseJSON, err := sendCommand("deleteMessage", b.Token, params)
	if err != nil {
		return err
	}

	var responseRecieved struct {
		Ok          bool
		Description string
	}

	err = json.Unmarshal(responseJSON, &responseRecieved)
	if err != nil {
		return err
	}

	if !responseRecieved.Ok {
		return fmt.Errorf("telebot: %s", responseRecieved.Description)
	}

	return nil
}

// SendPhoto sends a photo object to recipient.
//
// On success, photo object would be aliased to its copy on
// the Telegram servers, so sending the same photo object
// again, won't issue a new upload, but would make a use
// of existing file on Telegram servers.
func (b *Bot) SendPhoto(recipient Recipient, photo *Photo, options *SendOptions) error {
	params := map[string]string{
		"chat_id": recipient.Destination(),
		"caption": photo.Caption,
	}

	if options != nil {
		embedSendOptions(params, options)
	}

	var responseJSON []byte
	var err error

	if photo.Exists() {
		params["photo"] = photo.FileID
		responseJSON, err = sendCommand("sendPhoto", b.Token, params)
	} else {
		if len(photo.Url) > 0 {
			params["photo"] = photo.Url
			responseJSON, err = sendCommand("sendPhoto", b.Token, params)
		} else {
			responseJSON, err = sendFile("sendPhoto", b.Token, "photo",
				photo.filename, params)

		}

	}

	if err != nil {
		return err
	}

	var responseRecieved struct {
		Ok          bool
		Result      Message
		Description string
	}

	err = json.Unmarshal(responseJSON, &responseRecieved)
	if err != nil {
		return err
	}

	if !responseRecieved.Ok {
		return fmt.Errorf("telebot: %s", responseRecieved.Description)
	}

	thumbnails := &responseRecieved.Result.Photo
	filename := photo.filename
	photo.File = (*thumbnails)[len(*thumbnails) - 1].File
	photo.filename = filename

	return nil
}

// SendAudio sends an audio object to recipient.
//
// On success, audio object would be aliased to its copy on
// the Telegram servers, so sending the same audio object
// again, won't issue a new upload, but would make a use
// of existing file on Telegram servers.
func (b *Bot) SendAudio(recipient Recipient, audio *Audio, options *SendOptions) error {
	params := map[string]string{
		"chat_id": recipient.Destination(),
	}

	if options != nil {
		embedSendOptions(params, options)
	}

	var responseJSON []byte
	var err error

	if audio.Exists() {
		params["audio"] = audio.FileID
		responseJSON, err = sendCommand("sendAudio", b.Token, params)
	} else {
		responseJSON, err = sendFile("sendAudio", b.Token, "audio",
			audio.filename, params)
	}

	if err != nil {
		return err
	}

	var responseRecieved struct {
		Ok          bool
		Result      Message
		Description string
	}

	err = json.Unmarshal(responseJSON, &responseRecieved)
	if err != nil {
		return err
	}

	if !responseRecieved.Ok {
		return fmt.Errorf("telebot: %s", responseRecieved.Description)
	}

	filename := audio.filename
	*audio = responseRecieved.Result.Audio
	audio.filename = filename

	return nil
}

// SendDocument sends a general document object to recipient.
//
// On success, document object would be aliased to its copy on
// the Telegram servers, so sending the same document object
// again, won't issue a new upload, but would make a use
// of existing file on Telegram servers.
func (b *Bot) SendDocument(recipient Recipient, doc *Document, options *SendOptions) error {
	params := map[string]string{
		"chat_id": recipient.Destination(),
	}

	if options != nil {
		embedSendOptions(params, options)
	}

	var responseJSON []byte
	var err error

	if doc.Exists() {
		params["document"] = doc.FileID
		responseJSON, err = sendCommand("sendDocument", b.Token, params)
	} else {
		responseJSON, err = sendFile("sendDocument", b.Token, "document",
			doc.filename, params)
	}

	if err != nil {
		return err
	}

	var responseRecieved struct {
		Ok          bool
		Result      Message
		Description string
	}

	err = json.Unmarshal(responseJSON, &responseRecieved)
	if err != nil {
		return err
	}

	if !responseRecieved.Ok {
		return fmt.Errorf("telebot: %s", responseRecieved.Description)
	}

	filename := doc.filename
	*doc = responseRecieved.Result.Document
	doc.filename = filename

	return nil
}

// SendSticker sends a general document object to recipient.
//
// On success, sticker object would be aliased to its copy on
// the Telegram servers, so sending the same sticker object
// again, won't issue a new upload, but would make a use
// of existing file on Telegram servers.
func (b *Bot) SendSticker(recipient Recipient, sticker *Sticker, options *SendOptions) error {
	params := map[string]string{
		"chat_id": recipient.Destination(),
	}

	if options != nil {
		embedSendOptions(params, options)
	}

	var responseJSON []byte
	var err error

	if sticker.Exists() {
		params["sticker"] = sticker.FileID
		responseJSON, err = sendCommand("sendSticker", b.Token, params)
	} else {
		responseJSON, err = sendFile("sendSticker", b.Token, "sticker",
			sticker.filename, params)
	}

	if err != nil {
		return err
	}

	var responseRecieved struct {
		Ok          bool
		Result      Message
		Description string
	}

	err = json.Unmarshal(responseJSON, &responseRecieved)
	if err != nil {
		return err
	}

	if !responseRecieved.Ok {
		return fmt.Errorf("telebot: %s", responseRecieved.Description)
	}

	filename := sticker.filename
	*sticker = responseRecieved.Result.Sticker
	sticker.filename = filename

	return nil
}

// SendVideo sends a general document object to recipient.
//
// On success, video object would be aliased to its copy on
// the Telegram servers, so sending the same video object
// again, won't issue a new upload, but would make a use
// of existing file on Telegram servers.
func (b *Bot) SendVideo(recipient Recipient, video *Video, options *SendOptions) error {
	params := map[string]string{
		"chat_id": recipient.Destination(),
	}

	if options != nil {
		embedSendOptions(params, options)
	}

	var responseJSON []byte
	var err error

	if video.Exists() {
		params["video"] = video.FileID
		responseJSON, err = sendCommand("sendVideo", b.Token, params)
	} else {
		responseJSON, err = sendFile("sendVideo", b.Token, "video",
			video.filename, params)
	}

	if err != nil {
		return err
	}

	var responseRecieved struct {
		Ok          bool
		Result      Message
		Description string
	}

	err = json.Unmarshal(responseJSON, &responseRecieved)
	if err != nil {
		return err
	}

	if !responseRecieved.Ok {
		return fmt.Errorf("telebot: %s", responseRecieved.Description)
	}

	filename := video.filename
	*video = responseRecieved.Result.Video
	video.filename = filename

	return nil
}

// SendLocation sends a general document object to recipient.
//
// On success, video object would be aliased to its copy on
// the Telegram servers, so sending the same video object
// again, won't issue a new upload, but would make a use
// of existing file on Telegram servers.
func (b *Bot) SendLocation(recipient Recipient, geo *Location, options *SendOptions) error {
	params := map[string]string{
		"chat_id":   recipient.Destination(),
		"latitude":  fmt.Sprintf("%f", geo.Latitude),
		"longitude": fmt.Sprintf("%f", geo.Longitude),
	}

	if options != nil {
		embedSendOptions(params, options)
	}

	responseJSON, err := sendCommand("sendLocation", b.Token, params)
	if err != nil {
		return err
	}

	var responseRecieved struct {
		Ok          bool
		Result      Message
		Description string
	}

	err = json.Unmarshal(responseJSON, &responseRecieved)
	if err != nil {
		return err
	}

	if !responseRecieved.Ok {
		return fmt.Errorf("telebot: %s", responseRecieved.Description)
	}

	return nil
}

// SendVenue sends a venue object to recipient.
func (b *Bot) SendVenue(recipient Recipient, venue *Venue, options *SendOptions) error {
	params := map[string]string{
		"chat_id":   recipient.Destination(),
		"latitude":  fmt.Sprintf("%f", venue.Location.Latitude),
		"longitude": fmt.Sprintf("%f", venue.Location.Longitude),
		"title":     venue.Title,
		"address":   venue.Address}
	if venue.Foursquare_id != "" {
		params["foursquare_id"] = venue.Foursquare_id
	}

	if options != nil {
		embedSendOptions(params, options)
	}

	responseJSON, err := sendCommand("sendVenue", b.Token, params)
	if err != nil {
		return err
	}

	var responseRecieved struct {
		Ok          bool
		Result      Message
		Description string
	}

	err = json.Unmarshal(responseJSON, &responseRecieved)
	if err != nil {
		return err
	}

	if !responseRecieved.Ok {
		return fmt.Errorf("telebot: %s", responseRecieved.Description)
	}

	return nil
}

// SendChatAction updates a chat action for recipient.
//
// Chat action is a status message that recipient would see where
// you typically see "Harry is typing" status message. The only
// difference is that bots' chat actions live only for 5 seconds
// and die just once the client recieves a message from the bot.
//
// Currently, Telegram supports only a narrow range of possible
// actions, these are aligned as constants of this package.
func (b *Bot) SendChatAction(recipient Recipient, action string) error {
	params := map[string]string{
		"chat_id": recipient.Destination(),
		"action":  action,
	}

	responseJSON, err := sendCommand("sendChatAction", b.Token, params)
	if err != nil {
		return err
	}

	var responseRecieved struct {
		Ok          bool
		Description string
	}

	err = json.Unmarshal(responseJSON, &responseRecieved)
	if err != nil {
		return err
	}

	if !responseRecieved.Ok {
		return fmt.Errorf("telebot: %s", responseRecieved.Description)
	}

	return nil
}

// Respond publishes a set of responses for an inline query.
// This function is deprecated in favor of AnswerInlineQuery.
func (b *Bot) Respond(query Query, results []Result) error {
	params := map[string]string{
		"inline_query_id": query.ID,
	}

	if res, err := json.Marshal(results); err == nil {
		params["results"] = string(res)
	} else {
		return err
	}

	responseJSON, err := sendCommand("answerInlineQuery", b.Token, params)
	if err != nil {
		return err
	}

	var responseRecieved struct {
		Ok          bool
		Description string
	}

	err = json.Unmarshal(responseJSON, &responseRecieved)
	if err != nil {
		return err
	}

	if !responseRecieved.Ok {
		return fmt.Errorf("telebot: %s", responseRecieved.Description)
	}

	return nil
}

// AnswerInlineQuery sends a response for a given inline query. A query can
// only be responded to once, subsequent attempts to respond to the same query
// will result in an error.
func (b *Bot) AnswerInlineQuery(query *Query, response *QueryResponse) error {
	response.QueryID = query.ID

	responseJSON, err := sendCommand("answerInlineQuery", b.Token, response)
	if err != nil {
		return err
	}

	var responseRecieved struct {
		Ok          bool
		Description string
	}

	err = json.Unmarshal(responseJSON, &responseRecieved)
	if err != nil {
		return err
	}

	if !responseRecieved.Ok {
		return fmt.Errorf("telebot: %s", responseRecieved.Description)
	}

	return nil
}

// AnswerCallbackQuery sends a response for a given callback query. A callback can
// only be responded to once, subsequent attempts to respond to the same callback
// will result in an error.
func (b *Bot) AnswerCallbackQuery(callback *Callback, response *CallbackResponse) error {
	response.CallbackID = callback.ID

	responseJSON, err := sendCommand("answerCallbackQuery", b.Token, response)
	if err != nil {
		return err
	}

	var responseRecieved struct {
		Ok          bool
		Description string
	}

	err = json.Unmarshal(responseJSON, &responseRecieved)
	if err != nil {
		return err
	}

	if !responseRecieved.Ok {
		return fmt.Errorf("telebot: %s", responseRecieved.Description)
	}

	return nil
}

// Use this method to get a list of profile pictures for a user. Returns a UserProfilePhotos object.
// https://core.telegram.org/bots/api#getuserprofilephotos
func (b *Bot) GetUserProfilePhotos(userId string) (*[]UserProfilePhoto, error) {
	params := map[string]string{
		"user_id": userId,
	}

	var responseJSON []byte
	var err error

	responseJSON, err = sendCommand("getUserProfilePhotos", b.Token, params)
	if err != nil {
		return nil, err
	}

	var responseRecieved struct {
		Ok          bool
		Result      struct {
				    Photos [][]UserProfilePhoto
			    }
		Description string
	}

	err = json.Unmarshal(responseJSON, &responseRecieved)
	if err != nil {
		return nil, err
	}

	if !responseRecieved.Ok {
		return nil, fmt.Errorf("GetUserProfilePhotos: %s", responseRecieved.Description)
	}

	if len(responseRecieved.Result.Photos) > 0 {
		return &responseRecieved.Result.Photos[0], nil

	} else {
		return nil, errors.New("user has no avatar photo")
	}

}

// Use this method to get basic info about a file and prepare it for downloading
// https://core.telegram.org/bots/api#getfile
func (b *Bot) GetFile(fileId string) (*File, error) {
	params := map[string]string{
		"file_id": fileId,
	}

	var responseJSON []byte
	var err error

	responseJSON, err = sendCommand("getFile", b.Token, params)
	if err != nil {
		return nil, err
	}

	var responseRecieved struct {
		Ok          bool
		Result      struct {
				    FileId   string `json:"file_id"`
				    FileSize int `json:"file_size"`
				    FilePath string `json:"file_path"`
			    }
		Description string
	}

	err = json.Unmarshal(responseJSON, &responseRecieved)
	if err != nil {
		return nil, err
	}

	if !responseRecieved.Ok {
		return nil, fmt.Errorf("telebot: %s", responseRecieved.Description)
	}

	file := File{
		FileID: responseRecieved.Result.FileId,
		FileSize: responseRecieved.Result.FileSize,
		filename: responseRecieved.Result.FilePath,
	}

	return &file, nil
}

// SendPhoto sends a photo object to recipient.
func (b *Bot) SendPhotoAsLink(recipient Recipient, photoUrl string, options *SendOptions) error {
	params := map[string]string{
		"chat_id": recipient.Destination(),
		"caption": "!!!",
		//"caption": photo.Caption,
	}

	if options != nil {
		embedSendOptions(params, options)
	}

	var responseJSON []byte
	var err error

	params["photo"] = photoUrl
	responseJSON, err = sendCommand("sendPhoto", b.Token, params)

	if err != nil {
		return err
	}

	var responseRecieved struct {
		Ok          bool
		Result      Message
		Description string
	}

	err = json.Unmarshal(responseJSON, &responseRecieved)
	if err != nil {
		return err
	}

	if !responseRecieved.Ok {
		return fmt.Errorf("telebot: %s", responseRecieved.Description)
	}

	return nil
}

// SendPhoto sends a photo object to recipient.
func (b *Bot) SendVideoAsLink(recipient Recipient, videoUrl string, options *SendOptions) error {
	params := map[string]string{
		"chat_id": recipient.Destination(),
	}

	if options != nil {
		embedSendOptions(params, options)
	}

	var responseJSON []byte
	var err error

	params["video"] = videoUrl
	responseJSON, err = sendCommand("sendVideo", b.Token, params)

	if err != nil {
		return err
	}

	var responseRecieved struct {
		Ok          bool
		Result      Message
		Description string
	}

	err = json.Unmarshal(responseJSON, &responseRecieved)
	if err != nil {
		return err
	}

	if !responseRecieved.Ok {
		return fmt.Errorf("telebot: %s", responseRecieved.Description)
	}

	return nil
}

func (b *Bot) GetMe() (*BotInfo, error){
	var responseJSON []byte
	var err error

	responseJSON, err = sendCommand("getMe", b.Token, nil)

	if err != nil {
		return nil, err
	}

	var responseRecieved struct {
		Ok          bool
		Description string
		Result      BotInfo
	}

	err = json.Unmarshal(responseJSON, &responseRecieved)
	if err != nil {
		return nil, err
	}

	if !responseRecieved.Ok {
		return nil, fmt.Errorf("telebot: %s", responseRecieved.Description)
	}

	return &responseRecieved.Result, nil
}
