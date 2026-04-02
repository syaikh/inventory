package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"inventory/internal/models"
	"inventory/internal/store"
)

type Handler struct {
	store *store.Store
	// scanBus broadcasts scan events to SSE clients
	scanBus chan models.ScanEvent
}

func New(s *store.Store) *Handler {
	return &Handler{store: s, scanBus: make(chan models.ScanEvent, 32)}
}

func (h *Handler) ScanBus() chan<- models.ScanEvent { return h.scanBus }

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/scan", h.handleScan)
	mux.HandleFunc("/api/items", h.handleItems)
	mux.HandleFunc("/api/items/", h.handleItemByID)
	mux.HandleFunc("/api/transactions", h.handleTransactions)
	mux.HandleFunc("/api/categories", h.handleCategories)
	mux.HandleFunc("/api/stats", h.handleStats)
	mux.HandleFunc("/api/events", h.handleSSE)

	// Static files
	mux.Handle("/", http.FileServer(http.Dir("web/static")))

	return mux
}

// POST /api/scan  { barcode, mode, qty }
func (h *Handler) handleScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	var ev models.ScanEvent
	if err := json.NewDecoder(r.Body).Decode(&ev); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if ev.Barcode == "" {
		http.Error(w, "barcode required", 400)
		return
	}
	if ev.Qty == 0 {
		ev.Qty = 1
	}

	item, err := h.store.GetItem(ev.Barcode)
	if err != nil {
		jsonErr(w, err)
		return
	}

	// Auto-create item if not found
	if item == nil {
		item = &models.Item{Barcode: ev.Barcode, Name: "Unknown — " + ev.Barcode, Unit: "pcs"}
		if err := h.store.UpsertItem(item); err != nil {
			jsonErr(w, err)
			return
		}
		item, _ = h.store.GetItem(ev.Barcode)
	}

	delta := ev.Qty
	txType := "scan_in"
	if ev.Mode == "out" {
		delta = -ev.Qty
		txType = "scan_out"
	}

	updated, err := h.store.UpdateQuantity(ev.Barcode, delta)
	if err != nil {
		jsonErr(w, err)
		return
	}

	h.store.AddTransaction(&models.Transaction{
		Barcode:  ev.Barcode,
		ItemName: item.Name,
		Type:     txType,
		Quantity: ev.Qty,
	})

	// Broadcast to SSE clients
	select {
	case h.scanBus <- ev:
	default:
	}

	jsonOK(w, updated)
}

// GET /api/items?search=&category=
// POST /api/items (upsert)
func (h *Handler) handleItems(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		search := r.URL.Query().Get("search")
		category := r.URL.Query().Get("category")
		items, err := h.store.ListItems(search, category)
		if err != nil {
			jsonErr(w, err)
			return
		}
		if items == nil {
			items = []models.Item{}
		}
		jsonOK(w, items)

	case http.MethodPost:
		var it models.Item
		if err := json.NewDecoder(r.Body).Decode(&it); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		if err := h.store.UpsertItem(&it); err != nil {
			jsonErr(w, err)
			return
		}
		updated, _ := h.store.GetItem(it.Barcode)
		jsonOK(w, updated)

	default:
		http.Error(w, "method not allowed", 405)
	}
}

// GET /api/items/{id}
// PUT /api/items/{id}
// DELETE /api/items/{id}
func (h *Handler) handleItemByID(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/items/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", 400)
		return
	}

	switch r.Method {
	case http.MethodGet:
		item, err := h.store.GetItemByID(id)
		if err != nil {
			jsonErr(w, err)
			return
		}
		if item == nil {
			http.NotFound(w, r)
			return
		}
		jsonOK(w, item)

	case http.MethodPut:
		var it models.Item
		if err := json.NewDecoder(r.Body).Decode(&it); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		if err := h.store.UpsertItem(&it); err != nil {
			jsonErr(w, err)
			return
		}
		updated, _ := h.store.GetItemByID(id)
		jsonOK(w, updated)

	case http.MethodDelete:
		if err := h.store.DeleteItem(id); err != nil {
			jsonErr(w, err)
			return
		}
		jsonOK(w, map[string]bool{"deleted": true})

	default:
		http.Error(w, "method not allowed", 405)
	}
}

// GET /api/transactions?barcode=&limit=50
func (h *Handler) handleTransactions(w http.ResponseWriter, r *http.Request) {
	barcode := r.URL.Query().Get("barcode")
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil {
			limit = n
		}
	}
	txs, err := h.store.ListTransactions(barcode, limit)
	if err != nil {
		jsonErr(w, err)
		return
	}
	if txs == nil {
		txs = []models.Transaction{}
	}
	jsonOK(w, txs)
}

// GET /api/categories
func (h *Handler) handleCategories(w http.ResponseWriter, r *http.Request) {
	cats, err := h.store.Categories()
	if err != nil {
		jsonErr(w, err)
		return
	}
	if cats == nil {
		cats = []string{}
	}
	jsonOK(w, cats)
}

// GET /api/stats
func (h *Handler) handleStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.store.Stats()
	if err != nil {
		jsonErr(w, err)
		return
	}
	jsonOK(w, stats)
}

// GET /api/events  — Server-Sent Events for real-time scan notifications
func (h *Handler) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", 500)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	for {
		select {
		case ev := <-h.scanBus:
			data, _ := json.Marshal(ev)
			w.Write([]byte("data: " + string(data) + "\n\n"))
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func jsonOK(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func jsonErr(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(500)
	json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
