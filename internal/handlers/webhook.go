package handlers

import (
	"io"
	"log"
	"net/http"

	"mad-news-bot/internal/commands/apod"
	"mad-news-bot/internal/commands/covid"
	"mad-news-bot/internal/cache"
	"mad-news-bot/internal/config"
	"mad-news-bot/internal/telegram"
)

type WebhookHandler struct {
	Router *telegram.Router
	Redis  *cache.Redis
	Config config.Config
}

func (h *WebhookHandler) postReply(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "bad request"})
		return
	}
	result := h.Router.Handle(r.Context(), body)
	WriteJSON(w, http.StatusOK, result)
}

func (h *WebhookHandler) postHire(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "bad request"})
		return
	}
	result := h.Router.HandleHire(body)
	WriteJSON(w, http.StatusOK, result)
}

func (h *WebhookHandler) postNotify(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "bad request"})
		return
	}
	result := h.Router.HandleNotify(body)
	WriteJSON(w, http.StatusOK, result)
}

func (h *WebhookHandler) postWhatsApp(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err == nil {
		log.Printf("[WA] body: %s", r.FormValue("Body"))
	}
	w.Header().Set("Content-Type", "text/xml")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(h.Router.HandleWhatsApp()))
}

func (h *WebhookHandler) getCovid19(w http.ResponseWriter, r *http.Request) {
	msg, err := covid.Format(r.Context(), h.Redis)
	if err != nil {
		log.Printf("covid19 http: %v", err)
		WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "unavailable"})
		return
	}
	WriteJSON(w, http.StatusOK, msg)
}

func (h *WebhookHandler) getNASAAPOD(w http.ResponseWriter, r *http.Request) {
	res, err := apod.Get(r.Context(), h.Redis, h.Config.NASAAPODKey)
	if err != nil {
		log.Printf("nasaapod http: %v", err)
		WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "unavailable"})
		return
	}
	WriteJSON(w, http.StatusOK, res)
}

func NewMux(cfg config.Config, redis *cache.Redis, router *telegram.Router) http.Handler {
	h := &WebhookHandler{Router: router, Redis: redis, Config: cfg}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", Health)
	mux.HandleFunc("POST /api/webhooks/mad-news", h.postReply)
	mux.HandleFunc("POST /api/webhooks/mad-news-en", h.postReply)
	mux.HandleFunc("POST /api/webhooks/vovan", h.postReply)
	mux.HandleFunc("POST /api/webhooks/mad-news-wa", h.postWhatsApp)
	mux.HandleFunc("POST /api/webhooks/hire", h.postHire)
	mux.HandleFunc("POST /api/webhooks/notify", h.postNotify)
	mux.HandleFunc("GET /api/webhooks/covid19", h.getCovid19)
	mux.HandleFunc("GET /api/webhooks/nasaapod", h.getNASAAPOD)
	return WithCORS(mux)
}
