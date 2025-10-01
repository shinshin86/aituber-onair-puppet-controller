package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

type connectionMeta struct {
	clientType string
}

type hub struct {
	mu    sync.RWMutex
	conns map[*websocket.Conn]connectionMeta
}

func newHub() *hub {
	return &hub{
		conns: make(map[*websocket.Conn]connectionMeta),
	}
}

func (h *hub) add(conn *websocket.Conn, meta connectionMeta) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.conns[conn] = meta
}

func (h *hub) remove(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.conns, conn)
}

func (h *hub) stats() (total int, ui int) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, meta := range h.conns {
		total++
		if meta.clientType == "ui" {
			ui++
		}
	}

	return total, ui
}

func (h *hub) broadcast(payload any) {
	message, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[hub] failed to marshal payload: %v", err)
		return
	}

	h.mu.RLock()
	conns := make([]*websocket.Conn, 0, len(h.conns))
	for conn := range h.conns {
		conns = append(conns, conn)
	}
	h.mu.RUnlock()

	for _, conn := range conns {
		if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
			log.Printf("[direct-speech] send error: %v", err)
			h.remove(conn)
			_ = conn.Close()
		}
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func directSpeechHandler(h *hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		clientType := "external"
		if r.URL.Query().Get("client") == "ui" {
			clientType = "ui"
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("[direct-speech] upgrade failed: %v", err)
			return
		}

		log.Println("[direct-speech] client connected")
		h.add(conn, connectionMeta{clientType: clientType})

		go func() {
			defer func() {
				h.remove(conn)
				_ = conn.Close()
				log.Println("[direct-speech] client disconnected")
			}()

			for {
				if _, _, err := conn.ReadMessage(); err != nil {
					if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
						log.Printf("[direct-speech] socket error: %v", err)
					}
					return
				}
			}
		}()
	}
}

type triggerRequest struct {
	Text string `json:"text"`
}

func triggerHandler(h *hub) http.HandlerFunc {
	type response struct {
		Ok    bool   `json:"ok,omitempty"`
		Error string `json:"error,omitempty"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		defer r.Body.Close()

		var req triggerRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Printf("[trigger] invalid request: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(response{Error: "invalid JSON payload"})
			return
		}

		text := strings.TrimSpace(req.Text)
		if text == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(response{Error: "`text` field must be a non-empty string"})
			return
		}

		payload := map[string]string{
			"type": "chat",
			"text": text,
		}
		h.broadcast(payload)
		log.Printf("[trigger] broadcast: %+v", payload)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(response{Ok: true})
	}
}

func statusHandler(h *hub) http.HandlerFunc {
	type response struct {
		TotalConnections    int `json:"totalConnections"`
		UIConnections       int `json:"uiConnections"`
		ExternalConnections int `json:"externalConnections"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		total, ui := h.stats()
		resp := response{
			TotalConnections:    total,
			UIConnections:       ui,
			ExternalConnections: total - ui,
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Printf("[status] encode error: %v", err)
		}
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	http.ServeFile(w, r, "web/index.html")
}

func main() {
	hub := newHub()

	mux := http.NewServeMux()
	mux.HandleFunc("/direct-speech", directSpeechHandler(hub))
	mux.HandleFunc("/trigger", triggerHandler(hub))
	mux.HandleFunc("/status", statusHandler(hub))
	mux.Handle("/logo/", http.StripPrefix("/logo/", http.FileServer(http.Dir("logo"))))
	mux.HandleFunc("/", indexHandler)

	server := &http.Server{
		Addr:    ":9000",
		Handler: mux,
	}

	log.Println("AITuber OnAir Puppet Controller listening on http://localhost:9000")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}
