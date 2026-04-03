package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"

	"inventory/internal/models"
	"inventory/internal/service"
)

// Constants for HTTP handling
const (
	defaultTransactionLimit = 50
	contentTypeJSON         = "application/json"
	contentTypeSSE          = "text/event-stream"
)

// Handler handles HTTP requests and responses
type Handler struct {
	svc service.InventoryService
}

func New(svc service.InventoryService) *Handler {
	return &Handler{
		svc: svc,
	}
}

// Helper functions for common operations
func ensureNonNilSlice[T any](slice []T) []T {
	if slice == nil {
		return []T{}
	}
	return slice
}

func validateMethod(w http.ResponseWriter, r *http.Request, allowedMethod string) bool {
	if r.Method != allowedMethod {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return false
	}
	return true
}

func parseIDFromPath(path, prefix string) (int64, error) {
	idStr := strings.TrimPrefix(path, prefix)
	return strconv.ParseInt(idStr, 10, 64)
}

func parseLimitParam(r *http.Request, defaultLimit int) int {
	limitStr := r.URL.Query().Get("limit")
	if limitStr == "" {
		return defaultLimit
	}
	if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
		return limit
	}
	return defaultLimit
}

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
	staticDir := "web/static"
	if _, err := os.Stat("web/svelte-app/dist"); err == nil {
		staticDir = "web/svelte-app/dist"
	}
	mux.Handle("/", http.FileServer(http.Dir(staticDir)))

	return mux
}

// POST /api/scan  { barcode, mode, qty }
func (h *Handler) handleScan(w http.ResponseWriter, r *http.Request) {
	if !validateMethod(w, r, http.MethodPost) {
		return
	}

	var ev models.ScanEvent
	if err := json.NewDecoder(r.Body).Decode(&ev); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	if ev.Barcode == "" {
		http.Error(w, "barcode required", http.StatusBadRequest)
		return
	}

	if ev.Qty == 0 {
		ev.Qty = 1
	}

	// Process scan event via service
	updatedItem, err := h.svc.ProcessScan(ev)
	if err != nil {
		jsonErr(w, err)
		return
	}

	jsonOK(w, updatedItem)
}

// GET /api/items?search=&category=
// POST /api/items (upsert)
func (h *Handler) handleItems(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleGetItems(w, r)
	case http.MethodPost:
		h.handleCreateItem(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleGetItems(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	category := r.URL.Query().Get("category")

	items, err := h.svc.ListItems(search, category)
	if err != nil {
		jsonErr(w, err)
		return
	}

	jsonOK(w, ensureNonNilSlice(items))
}

func (h *Handler) handleCreateItem(w http.ResponseWriter, r *http.Request) {
	var item models.Item
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.svc.UpsertItem(&item); err != nil {
		jsonErr(w, err)
		return
	}

	// Return the updated/created item
	updated, err := h.svc.GetItem(item.Barcode)
	if err != nil {
		jsonErr(w, err)
		return
	}

	jsonOK(w, updated)
}

// GET /api/items/{id}
// PUT /api/items/{id}
// DELETE /api/items/{id}
func (h *Handler) handleItemByID(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromPath(r.URL.Path, "/api/items/")
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.handleGetItemByID(w, r, id)
	case http.MethodPut:
		h.handleUpdateItem(w, r, id)
	case http.MethodDelete:
		h.handleDeleteItem(w, r, id)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleGetItemByID(w http.ResponseWriter, r *http.Request, id int64) {
	item, err := h.svc.GetItemByID(id)
	if err != nil {
		jsonErr(w, err)
		return
	}
	if item == nil {
		http.NotFound(w, r)
		return
	}
	jsonOK(w, item)
}

func (h *Handler) handleUpdateItem(w http.ResponseWriter, r *http.Request, id int64) {
	var item models.Item
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.svc.UpsertItem(&item); err != nil {
		jsonErr(w, err)
		return
	}

	// Return the updated item
	updated, err := h.svc.GetItemByID(id)
	if err != nil {
		jsonErr(w, err)
		return
	}

	jsonOK(w, updated)
}

func (h *Handler) handleDeleteItem(w http.ResponseWriter, _ *http.Request, id int64) {
	if err := h.svc.DeleteItem(id); err != nil {
		jsonErr(w, err)
		return
	}
	jsonOK(w, map[string]bool{"deleted": true})
}

// GET /api/transactions?barcode=&limit=50
func (h *Handler) handleTransactions(w http.ResponseWriter, r *http.Request) {
	if !validateMethod(w, r, http.MethodGet) {
		return
	}

	barcode := r.URL.Query().Get("barcode")
	limit := parseLimitParam(r, defaultTransactionLimit)

	transactions, err := h.svc.ListTransactions(barcode, limit)
	if err != nil {
		jsonErr(w, err)
		return
	}

	jsonOK(w, ensureNonNilSlice(transactions))
}

// GET /api/categories
func (h *Handler) handleCategories(w http.ResponseWriter, r *http.Request) {
	if !validateMethod(w, r, http.MethodGet) {
		return
	}

	categories, err := h.svc.Categories()
	if err != nil {
		jsonErr(w, err)
		return
	}

	jsonOK(w, ensureNonNilSlice(categories))
}

// GET /api/stats
func (h *Handler) handleStats(w http.ResponseWriter, r *http.Request) {
	if !validateMethod(w, r, http.MethodGet) {
		return
	}

	stats, err := h.svc.Stats()
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
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ctx := r.Context()
	_ = r // suppress unused parameter warning
	for {
		select {
		case ev := <-h.svc.ScanBus():
			data, err := json.Marshal(ev)
			if err != nil {
				// Skip malformed events
				continue
			}
			w.Write([]byte("data: " + string(data) + "\n\n"))
			flusher.Flush()
		case <-ctx.Done():
			return
		}
	}
}

func jsonOK(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", contentTypeJSON)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(v)
}

func jsonErr(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", contentTypeJSON)
	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
