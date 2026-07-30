package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"mad-news-bot/internal/advicebridge"
	"mad-news-bot/internal/cache"
	"mad-news-bot/internal/chatlog"
	"mad-news-bot/internal/commands/apod"
	"mad-news-bot/internal/commands/caradvice"
	"mad-news-bot/internal/commands/covid"
	"mad-news-bot/internal/commands/currency"
	"mad-news-bot/internal/commands/madnews"
	"mad-news-bot/internal/commands/scope"
	"mad-news-bot/internal/config"
	"mad-news-bot/internal/oraclequeue"
)

type Update struct {
	Message *Message `json:"message"`
	Service string   `json:"service"`
}

type Message struct {
	Text string `json:"text"`
	Chat Chat   `json:"chat"`
	From *User  `json:"from"`
}

type Chat struct {
	ID       int64  `json:"id"`
	Title    string `json:"title"`
	Username string `json:"username"`
	Type     string `json:"type"`
}

type User struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

func chatLogEntry(msg *Message) chatlog.Entry {
	entry := chatlog.Entry{
		ChatID:       msg.Chat.ID,
		ChatTitle:    msg.Chat.Title,
		ChatUsername: msg.Chat.Username,
		ChatType:     msg.Chat.Type,
		Text:         msg.Text,
	}
	if msg.From != nil {
		entry.SenderID = msg.From.ID
		entry.SenderUsername = msg.From.Username
		entry.SenderFirstName = msg.From.FirstName
		entry.SenderLastName = msg.From.LastName
	}
	return entry
}

type Reply struct {
	Status  string `json:"status"`
	Code    int    `json:"code"`
	Message any    `json:"message,omitempty"`
}

type Router struct {
	cfg          config.Config
	tg           *Client
	redis        *cache.Redis
	oracleQueue  *oraclequeue.Queue
	adviceBridge *advicebridge.Store
	chatLog      *chatlog.Logger
}

func NewRouter(cfg config.Config, tg *Client, redis *cache.Redis, oracleQueue *oraclequeue.Queue, adviceBridge *advicebridge.Store, chatLog *chatlog.Logger) *Router {
	return &Router{cfg: cfg, tg: tg, redis: redis, oracleQueue: oracleQueue, adviceBridge: adviceBridge, chatLog: chatLog}
}

func (r *Router) Handle(ctx context.Context, body []byte) Reply {
	var update Update
	if err := json.Unmarshal(body, &update); err != nil {
		return Reply{Status: "success", Code: 0}
	}

	if update.Service == "updateCovid" {
		msg, err := covid.Format(ctx, r.redis)
		if err != nil {
			log.Printf("updateCovid: %v", err)
			return Reply{Status: "error", Code: 1}
		}
		return Reply{Status: "success", Code: 0, Message: msg}
	}

	if update.Service == "updateApod" {
		if err := apod.FetchAndStore(ctx, r.redis, r.cfg.NASAAPODKey); err != nil {
			log.Printf("updateApod: %v", err)
			return Reply{Status: "error", Code: 1}
		}
		return Reply{Status: "success", Code: 0}
	}

	if update.Message == nil || update.Message.Text == "" || update.Message.Chat.ID == 0 {
		return Reply{Status: "success", Code: 0}
	}

	text := NormalizeCommand(update.Message.Text, r.cfg.BotUsername)
	chatID := update.Message.Chat.ID
	entry := chatLogEntry(update.Message)
	log.Printf("[chat message] %s", chatlog.FormatLine(entry))
	if r.chatLog != nil {
		if err := r.chatLog.Log(entry); err != nil {
			log.Printf("chat log: %v", err)
		}
	}

	switch CommandName(text) {
	case "/prograscope":
		_ = r.tg.SendMessage(chatID, scope.Prograscope(entry.SenderID))
	case "/nasaapod":
		r.handleAPOD(ctx, chatID)
	case "/alcoscope":
		_ = r.tg.SendMessage(chatID, scope.Alcoscope(entry.SenderID))
	case "/covid19":
		msg, err := covid.Format(ctx, r.redis)
		if err != nil {
			log.Printf("covid19: %v", err)
			_ = r.tg.SendMessage(chatID, "Не удалось получить данные по COVID-19.")
		} else {
			_ = r.tg.SendMessage(chatID, msg)
		}
	case "/currency":
		args := strings.ToUpper(strings.TrimSpace(CommandArgs(text)))
		msg, err := currency.Format(ctx, args)
		if err != nil {
			log.Printf("currency: %v", err)
			_ = r.tg.SendMessage(chatID, "Не могу получить данные с cbr-xml-daily ;(")
		} else {
			_ = r.tg.SendMessage(chatID, msg)
		}
	case "/madnews":
		msg, err := madnews.Generate("ru")
		if err != nil {
			log.Printf("madnews: %v", err)
			_ = r.tg.SendMessage(chatID, "Не удалось сгенерировать новость.")
		} else {
			log.Printf("New madness: [%s]", msg)
			_ = r.tg.SendMessage(chatID, msg)
		}
	case "/carAdvice":
		_ = r.tg.SendMessage(chatID, caradvice.Next())
	case "/badadvice":
		r.handleBadAdvice(ctx, chatID, CommandArgs(text))
	}

	return Reply{Status: "success", Code: 0}
}

func (r *Router) handleAPOD(ctx context.Context, chatID int64) {
	res, err := apod.GetCached(ctx, r.redis)
	if err != nil {
		_ = r.tg.SendMessage(chatID, "Данные NASA APOD обновляются, попробуйте позже.")
		return
	}
	if res.MediaType == "video" || res.Photo == "" {
		if err := r.tg.SendMessage(chatID, res.Message); err != nil {
			log.Printf("apod send message: %v", err)
		}
		return
	}
	if err := r.tg.SendPhoto(chatID, res.Photo, res.Message); err != nil {
		log.Printf("apod send photo: %v", err)
	}
}

func (r *Router) handleBadAdvice(ctx context.Context, chatID int64, prompt string) {
	prompt = strings.TrimSpace(prompt)
	if len([]rune(prompt)) < 5 {
		_ = r.tg.SendMessage(chatID, "Напишите вопрос после команды: /badadvice купить ли мне ламборгини?")
		return
	}
	if r.oracleQueue == nil || r.adviceBridge == nil {
		_ = r.tg.SendMessage(chatID, "Сервис советов временно недоступен.")
		return
	}
	_ = r.tg.SendMessage(chatID, "Думаю над советом...")
	jobID, err := r.oracleQueue.Enqueue(ctx, prompt, "ru")
	if err != nil {
		log.Printf("badadvice enqueue: %v", err)
		_ = r.tg.SendMessage(chatID, "Не удалось поставить запрос в очередь.")
		return
	}
	if err := r.adviceBridge.Bind(ctx, jobID, chatID); err != nil {
		log.Printf("badadvice bind %s: %v", jobID, err)
		_ = r.tg.SendMessage(chatID, "Не удалось сохранить запрос.")
	}
}

func (r *Router) HandleNotify(body []byte) Reply {
	var payload struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return Reply{Status: "error", Code: 1}
	}
	text := strings.TrimSpace(payload.Text)
	if text == "" {
		return Reply{Status: "error", Code: 1}
	}
	chatID := int64(0)
	fmt.Sscanf(r.cfg.HireChatID, "%d", &chatID)
	if chatID == 0 {
		return Reply{Status: "error", Code: 1}
	}
	if err := r.tg.SendMessage(chatID, text); err != nil {
		log.Printf("notify: %v", err)
		return Reply{Status: "error", Code: 1}
	}
	return Reply{Status: "success", Code: 0}
}

func (r *Router) HandleHire(body []byte, clientIP string) Reply {
	var payload struct {
		Points  int `json:"points"`
		History []struct {
			Question string `json:"question"`
			Answer   string `json:"answer"`
		} `json:"history"`
	}
	if err := json.Unmarshal(body, &payload); err != nil || len(payload.History) == 0 {
		return Reply{Status: "error", Code: 1}
	}
	var b strings.Builder
	if clientIP != "" {
		fmt.Fprintf(&b, "IP: %s\n", clientIP)
	}
	fmt.Fprintf(&b, "Points: %d\n", payload.Points)
	for _, item := range payload.History {
		fmt.Fprintf(&b, "%s: %s\n", item.Question, item.Answer)
	}
	chatID := int64(0)
	fmt.Sscanf(r.cfg.HireChatID, "%d", &chatID)
	if chatID != 0 {
		_ = r.tg.SendMessage(chatID, b.String())
	}
	return Reply{Status: "success", Code: 0}
}

func (r *Router) HandleWhatsApp() string {
	msg, err := madnews.Generate("ru")
	if err != nil {
		log.Printf("madnews wa: %v", err)
		msg = "Не удалось сгенерировать новость."
	} else {
		log.Printf("New madness: [%s]", msg)
	}
	return twimlMessage(msg)
}

func twimlMessage(text string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?><Response><Message>%s</Message></Response>`, xmlEscape(text))
}

func xmlEscape(s string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		"'", "&apos;",
		"\"", "&quot;",
	)
	return replacer.Replace(s)
}
