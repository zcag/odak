package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/zcag/odak/internal/model"
	"github.com/zcag/odak/internal/parser"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (h *handler) ws(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token != h.cfg.APIKey {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	c := &client{conn: conn, send: make(chan []byte, 16)}
	h.hub.register <- c

	// writer goroutine
	go func() {
		defer conn.Close()
		for msg := range c.send {
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		}
	}()

	// reader / ping-pong loop
	conn.SetReadLimit(512)
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	go func() {
		for range ticker.C {
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}()
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			h.hub.unregister <- c
			return
		}
	}
}

func (h *handler) list(w http.ResponseWriter, r *http.Request) {
	f, err := h.store.Read()
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}

	items := f.Flat()
	if sec := r.URL.Query().Get("section"); sec != "" {
		var filtered []*model.Item
		for _, it := range items {
			if string(it.Section) == sec {
				filtered = append(filtered, it)
			}
		}
		items = filtered
	}
	if tag := r.URL.Query().Get("tag"); tag != "" {
		var filtered []*model.Item
		for _, it := range items {
			for _, t := range it.Tags {
				if t == tag {
					filtered = append(filtered, it)
					break
				}
			}
		}
		items = filtered
	}
	if pid := r.URL.Query().Get("parent_id"); pid != "" {
		var filtered []*model.Item
		for _, it := range items {
			if it.ParentID == pid {
				filtered = append(filtered, it)
			}
		}
		items = filtered
	}

	writeJSON(w, 200, items)
}

func (h *handler) create(w http.ResponseWriter, r *http.Request) {
	var item model.Item
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	if item.Text == "" {
		writeErr(w, 400, "text required")
		return
	}
	if item.Section == "" {
		item.Section = "Inbox"
	}

	// compute ID from content
	raw := buildRaw(&item)
	parsed := parser.ParseItem(raw, item.Section, item.Done, 0)
	item.ID = parsed.ID

	added, err := h.store.AddItem(&item)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	h.hub.Broadcast([]byte(`{"type":"reload"}`))
	writeJSON(w, 201, added)
}

func (h *handler) get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	f, err := h.store.Read()
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	item := f.ByID(id)
	if item == nil {
		writeErr(w, 404, "not found")
		return
	}
	writeJSON(w, 200, item)
}

func (h *handler) update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var patch model.Item
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	updated, err := h.store.UpdateItem(id, &patch)
	if err != nil {
		writeErr(w, 404, err.Error())
		return
	}
	h.hub.Broadcast([]byte(`{"type":"reload"}`))
	writeJSON(w, 200, updated)
}

func (h *handler) delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.store.DeleteItem(id); err != nil {
		writeErr(w, 404, err.Error())
		return
	}
	h.hub.Broadcast([]byte(`{"type":"reload"}`))
	w.WriteHeader(204)
}

func (h *handler) toggleDone(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	item, err := h.store.ToggleDone(id)
	if err != nil {
		writeErr(w, 404, err.Error())
		return
	}
	h.hub.Broadcast([]byte(`{"type":"reload"}`))
	writeJSON(w, 200, item)
}

func (h *handler) move(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		Section model.Section `json:"section"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	item, err := h.store.MoveItem(id, body.Section)
	if err != nil {
		writeErr(w, 404, err.Error())
		return
	}
	h.hub.Broadcast([]byte(`{"type":"reload"}`))
	writeJSON(w, 200, item)
}

func (h *handler) reorder(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Section string   `json:"section"`
		IDs     []string `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	if body.Section == "" || len(body.IDs) == 0 {
		writeErr(w, 400, "section and ids required")
		return
	}
	if err := h.store.ReorderItems(body.Section, body.IDs); err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	h.hub.Broadcast([]byte(`{"type":"reload"}`))
	w.WriteHeader(204)
}

func (h *handler) sections(w http.ResponseWriter, r *http.Request) {
	f, err := h.store.Read()
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	counts := make(map[string]int)
	for _, item := range f.Flat() {
		counts[item.Section]++
	}
	type sectionInfo struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}
	out := make([]sectionInfo, 0, len(f.SectionOrder))
	for _, s := range f.SectionOrder {
		out = append(out, sectionInfo{Name: s, Count: counts[s]})
	}
	writeJSON(w, 200, out)
}

func (h *handler) getRaw(w http.ResponseWriter, r *http.Request) {
	content, err := h.store.ReadRaw()
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte(content))
}

func (h *handler) putRaw(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	if err := h.store.WriteRaw(string(body)); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	h.hub.Broadcast([]byte(`{"type":"reload"}`))
	w.WriteHeader(204)
}

// buildRaw reconstructs the raw item line for ID computation.
func buildRaw(item *model.Item) string {
	var sb strings.Builder
	for _, tag := range item.Tags {
		sb.WriteString("[t:")
		sb.WriteString(tag)
		sb.WriteString("] ")
	}
	if item.Urgent {
		sb.WriteString("[!] ")
	}
	if item.Deadline != "" {
		sb.WriteString("[d:")
		sb.WriteString(item.Deadline)
		sb.WriteString("] ")
	}
	if item.Trigger != "" {
		sb.WriteString("[w:")
		sb.WriteString(item.Trigger)
		sb.WriteString("] ")
	}
	sb.WriteString(item.Text)
	return strings.TrimSpace(sb.String())
}
