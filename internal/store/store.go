package store

import (
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"inventory/internal/models"

	_ "github.com/lib/pq"
)

// SQL query constants to avoid duplication and improve maintainability
const (
	// Item field selection - used in multiple queries
	itemSelectFields = `id, barcode, name, sku, category, quantity, unit, price, location, created_at, updated_at`

	// Transaction field selection
	transactionSelectFields = `id, barcode, item_name, type, quantity, note, created_at`

	// Item table name
	itemsTable = `items`

	// Transaction table name
	transactionsTable = `transactions`
)

type Store struct {
	db *sql.DB
}

func New(dsn string) (*Store, error) {
	db, err := sql.Open("postgres", dsn)
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
			id         SERIAL PRIMARY KEY,
			barcode    VARCHAR(255) UNIQUE NOT NULL,
			name       VARCHAR(255) NOT NULL DEFAULT '',
			sku        VARCHAR(255) NOT NULL DEFAULT '',
			category   VARCHAR(255) NOT NULL DEFAULT '',
			quantity   INTEGER NOT NULL DEFAULT 0,
			unit       VARCHAR(50) NOT NULL DEFAULT 'pcs',
			price      DECIMAL(10,2) NOT NULL DEFAULT 0,
			location   VARCHAR(255) NOT NULL DEFAULT '',
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		);
		CREATE TABLE IF NOT EXISTS transactions (
			id         SERIAL PRIMARY KEY,
			barcode    VARCHAR(255) NOT NULL,
			item_name  VARCHAR(255) NOT NULL DEFAULT '',
			type       VARCHAR(50) NOT NULL,
			quantity   INTEGER NOT NULL,
			note       TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMP NOT NULL
		);
	`)
	return err
}

func (s *Store) Close() { s.db.Close() }

// Items

func (s *Store) GetItem(barcode string) (*models.Item, error) {
	row := s.db.QueryRow(
		fmt.Sprintf(`SELECT %s FROM %s WHERE barcode = $1`, itemSelectFields, itemsTable), barcode)
	return scanItem(row)
}

func (s *Store) GetItemByID(id int64) (*models.Item, error) {
	row := s.db.QueryRow(
		fmt.Sprintf(`SELECT %s FROM %s WHERE id = $1`, itemSelectFields, itemsTable), id)
	return scanItem(row)
}

func scanItem(row *sql.Row) (*models.Item, error) {
	var it models.Item
	err := row.Scan(&it.ID, &it.Barcode, &it.Name, &it.SKU, &it.Category,
		&it.Quantity, &it.Unit, &it.Price, &it.Location, &it.CreatedAt, &it.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil // Returns (nil, nil) for no-rows-found, not an error
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan item: %w", err)
	}
	return &it, nil
}

func (s *Store) UpsertItem(it *models.Item) error {
	now := time.Now()
	_, err := s.db.Exec(`
		INSERT INTO items (barcode, name, sku, category, quantity, unit, price, location, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
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
		`UPDATE items SET quantity = quantity + $1, updated_at = $2 WHERE barcode = $3`,
		delta, time.Now(), barcode)
	if err != nil {
		return nil, err
	}
	return s.GetItem(barcode)
}

func (s *Store) DeleteItem(id int64) error {
	_, err := s.db.Exec(`DELETE FROM items WHERE id = $1`, id)
	return err
}

func (s *Store) ListItems(search, category string) ([]models.Item, error) {
	q := fmt.Sprintf(`SELECT %s FROM %s WHERE 1=1`, itemSelectFields, itemsTable)
	args := []any{}
	if search != "" {
		q += ` AND (name ILIKE $1 OR barcode ILIKE $2 OR sku ILIKE $3)`
		like := "%" + search + "%"
		args = append(args, like, like, like)
	}
	if category != "" {
		q += fmt.Sprintf(` AND category = $%d`, len(args)+1)
		args = append(args, category)
	}
	q += ` ORDER BY name`
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query items: %w", err)
	}
	defer rows.Close()
	var items []models.Item
	for rows.Next() {
		var it models.Item
		err := rows.Scan(&it.ID, &it.Barcode, &it.Name, &it.SKU, &it.Category,
			&it.Quantity, &it.Unit, &it.Price, &it.Location, &it.CreatedAt, &it.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan item row: %w", err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating item rows: %w", err)
	}
	return items, nil
}

func (s *Store) Categories() ([]string, error) {
	rows, err := s.db.Query(fmt.Sprintf(`SELECT DISTINCT category FROM %s WHERE category != '' ORDER BY category`, itemsTable))
	if err != nil {
		return nil, fmt.Errorf("failed to query categories: %w", err)
	}
	defer rows.Close()
	var cats []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, fmt.Errorf("failed to scan category row: %w", err)
		}
		cats = append(cats, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating category rows: %w", err)
	}
	return cats, nil
}

// Transactions

func (s *Store) AddTransaction(t *models.Transaction) error {
	_, err := s.db.Exec(
		`INSERT INTO transactions (barcode, item_name, type, quantity, note, created_at) VALUES ($1,$2,$3,$4,$5,$6)`,
		t.Barcode, t.ItemName, t.Type, t.Quantity, t.Note, time.Now())
	return err
}

func (s *Store) ListTransactions(barcode string, limit int) ([]models.Transaction, error) {
	q := fmt.Sprintf(`SELECT %s FROM %s`, transactionSelectFields, transactionsTable)
	args := []any{}
	if barcode != "" {
		q += ` WHERE barcode = $1`
		args = append(args, barcode)
	}
	q += ` ORDER BY created_at DESC`
	if limit > 0 {
		q += ` LIMIT $` + strconv.Itoa(len(args)+1)
		args = append(args, limit)
	}
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query transactions: %w", err)
	}
	defer rows.Close()
	var txs []models.Transaction
	for rows.Next() {
		var t models.Transaction
		err := rows.Scan(&t.ID, &t.Barcode, &t.ItemName, &t.Type, &t.Quantity, &t.Note, &t.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction row: %w", err)
		}
		txs = append(txs, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating transaction rows: %w", err)
	}
	return txs, nil
}

func (s *Store) Stats() (map[string]any, error) {
	var totalItems, totalUnits, outOfStock, scansToday, totalCategories int64

	// Single query to get all stats - fixes N+1 query problem
	query := `
		SELECT
			(SELECT COUNT(*) FROM items) as total_items,
			(SELECT COALESCE(SUM(quantity), 0) FROM items) as total_units,
			(SELECT COUNT(*) FROM items WHERE quantity = 0) as out_of_stock,
			(SELECT COUNT(*) FROM transactions WHERE DATE(created_at) = CURRENT_DATE) as scans_today,
			(SELECT COUNT(DISTINCT category) FROM items WHERE category != '') as total_categories
	`
	err := s.db.QueryRow(query).Scan(&totalItems, &totalUnits, &outOfStock, &scansToday, &totalCategories)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	stats := map[string]any{
		"total_items":      totalItems,
		"total_units":      totalUnits,
		"out_of_stock":     outOfStock,
		"scans_today":      scansToday,
		"total_categories": totalCategories,
	}
	return stats, nil
}
