package handlers

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
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
	mux.HandleFunc("/api/products", h.handleProducts)
	mux.HandleFunc("/api/products/upload-csv", h.handleCSVUpload)
	mux.HandleFunc("/api/products/", h.handleProductByID)
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
	updatedProduct, err := h.svc.ProcessScan(ev)
	if err != nil {
		jsonErr(w, err)
		return
	}

	jsonOK(w, updatedProduct)
}

// GET /api/products?search=&category=
// POST /api/products (upsert)
func (h *Handler) handleProducts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleGetProducts(w, r)
	case http.MethodPost:
		h.handleCreateProduct(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleGetProducts(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	category := r.URL.Query().Get("category")

	products, err := h.svc.ListProducts(search, category)
	if err != nil {
		jsonErr(w, err)
		return
	}

	jsonOK(w, ensureNonNilSlice(products))
}

func (h *Handler) handleCreateProduct(w http.ResponseWriter, r *http.Request) {
	var product models.Product
	if err := json.NewDecoder(r.Body).Decode(&product); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.svc.UpsertProduct(&product); err != nil {
		jsonErr(w, err)
		return
	}

	// For creating, we use the primary barcode to fetch if we don't have ID back from UpsertProduct (though Upsert updates ID)
	var updated *models.Product
	var errFetch error

	if product.ID != 0 {
		updated, errFetch = h.svc.GetProductByID(product.ID)
	} else if len(product.Barcodes) > 0 {
		updated, errFetch = h.svc.GetProduct(product.Barcodes[0])
	}
	
	if errFetch != nil || updated == nil {
		jsonErr(w, fmt.Errorf("failed to reload product after creation"))
		return
	}

	jsonOK(w, updated)
}

// GET /api/products/{id}
// PUT /api/products/{id}
// DELETE /api/products/{id}
func (h *Handler) handleProductByID(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromPath(r.URL.Path, "/api/products/")
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.handleGetProductByID(w, r, id)
	case http.MethodPut:
		h.handleUpdateProduct(w, r, id)
	case http.MethodDelete:
		h.handleDeleteProduct(w, r, id)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleGetProductByID(w http.ResponseWriter, r *http.Request, id int64) {
	product, err := h.svc.GetProductByID(id)
	if err != nil {
		jsonErr(w, err)
		return
	}
	if product == nil {
		http.NotFound(w, r)
		return
	}
	jsonOK(w, product)
}

func (h *Handler) handleUpdateProduct(w http.ResponseWriter, r *http.Request, id int64) {
	var product models.Product
	if err := json.NewDecoder(r.Body).Decode(&product); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	
	product.ID = id // ensure we update the correct ID

	if err := h.svc.UpsertProduct(&product); err != nil {
		jsonErr(w, err)
		return
	}

	// Return the updated product
	updated, err := h.svc.GetProductByID(id)
	if err != nil {
		jsonErr(w, err)
		return
	}

	jsonOK(w, updated)
}

func (h *Handler) handleDeleteProduct(w http.ResponseWriter, _ *http.Request, id int64) {
	if err := h.svc.DeleteProduct(id); err != nil {
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

// POST /api/products/upload-csv
func (h *Handler) handleCSVUpload(w http.ResponseWriter, r *http.Request) {
	if !validateMethod(w, r, http.MethodPost) {
		return
	}

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "missing file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	// Read header
	_, err = reader.Read()
	if err != nil {
		http.Error(w, "failed to read header", http.StatusBadRequest)
		return
	}

	var successCount int
	var errorsList []string

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue // skip bad lines
		}

		if len(record) < 8 {
			errorsList = append(errorsList, fmt.Sprintf("invalid row format: %v", record))
			continue
		}

		name := strings.TrimSpace(record[0])
		if name == "" {
			continue // skip empty
		}
		category := strings.TrimSpace(record[1])
		price, _ := strconv.ParseFloat(strings.TrimSpace(record[2]), 64)
		qty, _ := strconv.Atoi(strings.TrimSpace(record[3]))
		sku := strings.TrimSpace(record[4])
		unit := strings.TrimSpace(record[5])
		if unit == "" {
			unit = "pcs"
		}
		location := strings.TrimSpace(record[6])
		barcodesStr := strings.TrimSpace(record[7])
		
		var barcodes []string
		if barcodesStr != "" {
			parts := strings.Split(barcodesStr, ",")
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if p != "" {
					barcodes = append(barcodes, p)
				}
			}
		}

		product := &models.Product{
			Name:     name,
			Category: category,
			Price:    price,
			Quantity: qty,
			SKU:      sku,
			Unit:     unit,
			Location: location,
			Barcodes: barcodes,
		}

		if err := h.svc.UpsertProduct(product); err != nil {
			errorsList = append(errorsList, fmt.Sprintf("failed to save product %s: %v", name, err))
		} else {
			successCount++
		}
	}

	jsonOK(w, map[string]any{
		"success": successCount,
		"errors":  errorsList,
	})
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
