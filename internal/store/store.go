package store

import (
	"database/sql"
	"fmt"
	"time"

	"inventory/internal/models"

	_ "github.com/mattn/go-sqlite3"
)

type Store struct {
	db *sql.DB
}

func New(path string) (*Store, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS items (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			barcode    TEXT    UNIQUE NOT NULL,
			name       TEXT    NOT NULL DEFAULT '',
			sku        TEXT    NOT NULL DEFAULT '',
			category   TEXT    NOT NULL DEFAULT '',
			quantity   INTEGER NOT NULL DEFAULT 0,
			unit       TEXT    NOT NULL DEFAULT 'pcs',
			price      REAL    NOT NULL DEFAULT 0,
			location   TEXT    NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		);
		CREATE TABLE IF NOT EXISTS transactions (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			barcode    TEXT    NOT NULL,
			item_name  TEXT    NOT NULL DEFAULT '',
			type       TEXT    NOT NULL,
			quantity   INTEGER NOT NULL,
			note       TEXT    NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL
		);
	`)
	return err
}

func (s *Store) Close() { s.db.Close() }

// Items

func (s *Store) GetItem(barcode string) (*models.Item, error) {
	row := s.db.QueryRow(
		`SELECT id, barcode, name, sku, category, quantity, unit, price, location, created_at, updated_at
		 FROM items WHERE barcode = ?`, barcode)
	return scanItem(row)
}

func (s *Store) GetItemByID(id int64) (*models.Item, error) {
	row := s.db.QueryRow(
		`SELECT id, barcode, name, sku, category, quantity, unit, price, location, created_at, updated_at
		 FROM items WHERE id = ?`, id)
	return scanItem(row)
}

func scanItem(row *sql.Row) (*models.Item, error) {
	var it models.Item
	err := row.Scan(&it.ID, &it.Barcode, &it.Name, &it.SKU, &it.Category,
		&it.Quantity, &it.Unit, &it.Price, &it.Location, &it.CreatedAt, &it.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &it, err
}

func (s *Store) UpsertItem(it *models.Item) error {
	now := time.Now()
	_, err := s.db.Exec(`
		INSERT INTO items (barcode, name, sku, category, quantity, unit, price, location, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(barcode) DO UPDATE SET
			name=excluded.name, sku=excluded.sku, category=excluded.category,
			unit=excluded.unit, price=excluded.price, location=excluded.location,
			updated_at=excluded.updated_at`,
		it.Barcode, it.Name, it.SKU, it.Category, it.Quantity,
		it.Unit, it.Price, it.Location, now, now)
	return err
}

func (s *Store) UpdateQuantity(barcode string, delta int) (*models.Item, error) {
	_, err := s.db.Exec(
		`UPDATE items SET quantity = quantity + ?, updated_at = ? WHERE barcode = ?`,
		delta, time.Now(), barcode)
	if err != nil {
		return nil, err
	}
	return s.GetItem(barcode)
}

func (s *Store) DeleteItem(id int64) error {
	_, err := s.db.Exec(`DELETE FROM items WHERE id = ?`, id)
	return err
}

func (s *Store) ListItems(search, category string) ([]models.Item, error) {
	q := `SELECT id, barcode, name, sku, category, quantity, unit, price, location, created_at, updated_at
	      FROM items WHERE 1=1`
	args := []any{}
	if search != "" {
		q += ` AND (name LIKE ? OR barcode LIKE ? OR sku LIKE ?)`
		like := "%" + search + "%"
		args = append(args, like, like, like)
	}
	if category != "" {
		q += ` AND category = ?`
		args = append(args, category)
	}
	q += ` ORDER BY name`
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []models.Item
	for rows.Next() {
		var it models.Item
		rows.Scan(&it.ID, &it.Barcode, &it.Name, &it.SKU, &it.Category,
			&it.Quantity, &it.Unit, &it.Price, &it.Location, &it.CreatedAt, &it.UpdatedAt)
		items = append(items, it)
	}
	return items, nil
}

func (s *Store) Categories() ([]string, error) {
	rows, err := s.db.Query(`SELECT DISTINCT category FROM items WHERE category != '' ORDER BY category`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var cats []string
	for rows.Next() {
		var c string
		rows.Scan(&c)
		cats = append(cats, c)
	}
	return cats, nil
}

// Transactions

func (s *Store) AddTransaction(t *models.Transaction) error {
	_, err := s.db.Exec(
		`INSERT INTO transactions (barcode, item_name, type, quantity, note, created_at) VALUES (?,?,?,?,?,?)`,
		t.Barcode, t.ItemName, t.Type, t.Quantity, t.Note, time.Now())
	return err
}

func (s *Store) ListTransactions(barcode string, limit int) ([]models.Transaction, error) {
	q := `SELECT id, barcode, item_name, type, quantity, note, created_at FROM transactions`
	args := []any{}
	if barcode != "" {
		q += ` WHERE barcode = ?`
		args = append(args, barcode)
	}
	q += ` ORDER BY created_at DESC`
	if limit > 0 {
		q += fmt.Sprintf(` LIMIT %d`, limit)
	}
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var txs []models.Transaction
	for rows.Next() {
		var t models.Transaction
		rows.Scan(&t.ID, &t.Barcode, &t.ItemName, &t.Type, &t.Quantity, &t.Note, &t.CreatedAt)
		txs = append(txs, t)
	}
	return txs, nil
}

func (s *Store) Stats() (map[string]any, error) {
	var totalItems int64
	var totalUnits int64
	var outOfStock int64
	var scansToday int64

	if err := s.db.QueryRow(`SELECT COUNT(*) FROM items`).Scan(&totalItems); err != nil {
		return nil, err
	}
	if err := s.db.QueryRow(`SELECT COALESCE(SUM(quantity), 0) FROM items`).Scan(&totalUnits); err != nil {
		return nil, err
	}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM items WHERE quantity = 0`).Scan(&outOfStock); err != nil {
		return nil, err
	}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM transactions WHERE DATE(created_at) = DATE('now')`).Scan(&scansToday); err != nil {
		return nil, err
	}

	stats := map[string]any{
		"total_items":  totalItems,
		"total_units":  totalUnits,
		"out_of_stock": outOfStock,
		"scans_today":  scansToday,
	}
	return stats, nil
}
