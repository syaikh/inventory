package models

import "time"

type Item struct {
	ID        int64     `json:"id"`
	Barcode   string    `json:"barcode"`
	Name      string    `json:"name"`
	SKU       string    `json:"sku"`
	Category  string    `json:"category"`
	Quantity  int       `json:"quantity"`
	Unit      string    `json:"unit"`
	Price     float64   `json:"price"`
	Location  string    `json:"location"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Transaction struct {
	ID        int64     `json:"id"`
	Barcode   string    `json:"barcode"`
	ItemName  string    `json:"item_name"`
	Type      string    `json:"type"` // "scan_in", "scan_out", "adjust"
	Quantity  int       `json:"quantity"`
	Note      string    `json:"note"`
	CreatedAt time.Time `json:"created_at"`
}

type ScanEvent struct {
	Barcode string `json:"barcode"`
	Mode    string `json:"mode"` // "in" or "out"
	Qty     int    `json:"qty"`
}
