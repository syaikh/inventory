package store

import (
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"inventory/internal/models"

	"github.com/lib/pq"
)

const (
	productSelectFields = `p.id, p.name, p.sku, p.category, p.quantity, p.unit, p.price, p.location, p.created_at, p.updated_at, COALESCE(array_remove(array_agg(b.barcode), NULL), '{}') as barcodes`
	
	transactionSelectFields = `id, barcode, product_name, type, quantity, note, created_at`
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
	// Dropping old tables to recreate fresh structure as user approved
	_, err := s.db.Exec(`
		DROP TABLE IF EXISTS transactions;
		DROP TABLE IF EXISTS product_barcodes;
		DROP TABLE IF EXISTS products;
		DROP TABLE IF EXISTS items;

		CREATE TABLE products (
			id         SERIAL PRIMARY KEY,
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

		CREATE TABLE product_barcodes (
			id         SERIAL PRIMARY KEY,
			product_id INTEGER REFERENCES products(id) ON DELETE CASCADE,
			barcode    VARCHAR(255) UNIQUE NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_barcode ON product_barcodes(barcode);

		CREATE TABLE transactions (
			id           SERIAL PRIMARY KEY,
			barcode      VARCHAR(255) NOT NULL,
			product_name VARCHAR(255) NOT NULL DEFAULT '',
			type         VARCHAR(50) NOT NULL,
			quantity     INTEGER NOT NULL,
			note         TEXT NOT NULL DEFAULT '',
			created_at   TIMESTAMP NOT NULL
		);
	`)
	return err
}

func (s *Store) Close() { s.db.Close() }

// Products

func (s *Store) GetProduct(barcode string) (*models.Product, error) {
	q := fmt.Sprintf(`
		SELECT %s 
		FROM products p 
		LEFT JOIN product_barcodes b ON p.id = b.product_id 
		WHERE p.id = (SELECT product_id FROM product_barcodes WHERE barcode = $1)
		GROUP BY p.id`, productSelectFields)
	
	row := s.db.QueryRow(q, barcode)
	return scanProduct(row)
}

func (s *Store) GetProductByID(id int64) (*models.Product, error) {
	q := fmt.Sprintf(`
		SELECT %s 
		FROM products p 
		LEFT JOIN product_barcodes b ON p.id = b.product_id 
		WHERE p.id = $1
		GROUP BY p.id`, productSelectFields)
		
	row := s.db.QueryRow(q, id)
	return scanProduct(row)
}

func scanProduct(row *sql.Row) (*models.Product, error) {
	var it models.Product
	err := row.Scan(&it.ID, &it.Name, &it.SKU, &it.Category,
		&it.Quantity, &it.Unit, &it.Price, &it.Location, &it.CreatedAt, &it.UpdatedAt, pq.Array(&it.Barcodes))
	if err == sql.ErrNoRows {
		return nil, nil // Returns (nil, nil) for no-rows-found, not an error
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan product: %w", err)
	}
	return &it, nil
}

func (s *Store) UpsertProduct(it *models.Product) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now()
	
	if it.ID == 0 {
		// Insert new product
		err = tx.QueryRow(`
			INSERT INTO products (name, sku, category, quantity, unit, price, location, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`,
			it.Name, it.SKU, it.Category, it.Quantity, it.Unit, it.Price, it.Location, now, now).Scan(&it.ID)
		if err != nil {
			return err
		}
	} else {
		// Update existing product
		_, err = tx.Exec(`
			UPDATE products SET 
				name=$1, sku=$2, category=$3, quantity=$4, unit=$5, price=$6, location=$7, updated_at=$8
			WHERE id=$9`,
			it.Name, it.SKU, it.Category, it.Quantity, it.Unit, it.Price, it.Location, now, it.ID)
		if err != nil {
			return err
		}
	}

	// For barcodes, clear & insert since arrays are small
	_, err = tx.Exec(`DELETE FROM product_barcodes WHERE product_id = $1`, it.ID)
	if err != nil {
		return err
	}

	for _, barcode := range it.Barcodes {
		if barcode == "" {
			continue
		}
		// Insert ignoring conflicts if multiple products somehow tried to claim same barcode
		_, err = tx.Exec(`
			INSERT INTO product_barcodes (product_id, barcode) 
			VALUES ($1, $2) ON CONFLICT (barcode) DO NOTHING`, it.ID, barcode)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *Store) UpdateQuantity(barcode string, delta int) (*models.Product, error) {
	// First get the product id via barcode
	var prodID int64
	err := s.db.QueryRow(`SELECT product_id FROM product_barcodes WHERE barcode = $1`, barcode).Scan(&prodID)
	if err != nil {
		return nil, fmt.Errorf("barcode not found: %w", err)
	}

	_, err = s.db.Exec(`UPDATE products SET quantity = quantity + $1, updated_at = $2 WHERE id = $3`, delta, time.Now(), prodID)
	if err != nil {
		return nil, err
	}
	
	return s.GetProductByID(prodID)
}

func (s *Store) DeleteProduct(id int64) error {
	_, err := s.db.Exec(`DELETE FROM products WHERE id = $1`, id)
	return err
}

func (s *Store) ListProducts(search, category string) ([]models.Product, error) {
	q := fmt.Sprintf(`SELECT %s FROM products p LEFT JOIN product_barcodes b ON p.id = b.product_id WHERE 1=1`, productSelectFields)
	args := []any{}
	if search != "" {
		q += ` AND (p.name ILIKE $1 OR p.sku ILIKE $2 OR p.id IN (SELECT product_id FROM product_barcodes WHERE barcode ILIKE $3))`
		like := "%" + search + "%"
		args = append(args, like, like, like)
	}
	if category != "" {
		q += fmt.Sprintf(` AND p.category = $%d`, len(args)+1)
		args = append(args, category)
	}
	q += ` GROUP BY p.id ORDER BY p.name`
	
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query products: %w", err)
	}
	defer rows.Close()
	
	var products []models.Product
	for rows.Next() {
		var it models.Product
		err := rows.Scan(&it.ID, &it.Name, &it.SKU, &it.Category,
			&it.Quantity, &it.Unit, &it.Price, &it.Location, &it.CreatedAt, &it.UpdatedAt, pq.Array(&it.Barcodes))
		if err != nil {
			return nil, fmt.Errorf("failed to scan product row: %w", err)
		}
		products = append(products, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating product rows: %w", err)
	}
	return products, nil
}

func (s *Store) Categories() ([]string, error) {
	rows, err := s.db.Query(`SELECT DISTINCT category FROM products WHERE category != '' ORDER BY category`)
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
	return cats, rows.Err()
}

// Transactions

func (s *Store) AddTransaction(t *models.Transaction) error {
	_, err := s.db.Exec(
		`INSERT INTO transactions (barcode, product_name, type, quantity, note, created_at) VALUES ($1,$2,$3,$4,$5,$6)`,
		t.Barcode, t.ProductName, t.Type, t.Quantity, t.Note, time.Now())
	return err
}

func (s *Store) ListTransactions(barcode string, limit int) ([]models.Transaction, error) {
	q := fmt.Sprintf(`SELECT %s FROM transactions`, transactionSelectFields)
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
		err := rows.Scan(&t.ID, &t.Barcode, &t.ProductName, &t.Type, &t.Quantity, &t.Note, &t.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction row: %w", err)
		}
		txs = append(txs, t)
	}
	return txs, rows.Err()
}

func (s *Store) Stats() (map[string]any, error) {
	var totalProducts, totalUnits, outOfStock, scansToday, totalCategories int64

	query := `
		SELECT
			(SELECT COUNT(*) FROM products) as total_products,
			(SELECT COALESCE(SUM(quantity), 0) FROM products) as total_units,
			(SELECT COUNT(*) FROM products WHERE quantity <= 0) as out_of_stock,
			(SELECT COUNT(*) FROM transactions WHERE DATE(created_at) = CURRENT_DATE) as scans_today,
			(SELECT COUNT(DISTINCT category) FROM products WHERE category != '') as total_categories
	`
	err := s.db.QueryRow(query).Scan(&totalProducts, &totalUnits, &outOfStock, &scansToday, &totalCategories)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	stats := map[string]any{
		"total_products":   totalProducts,
		"total_units":      totalUnits,
		"out_of_stock":     outOfStock,
		"scans_today":      scansToday,
		"total_categories": totalCategories,
	}
	return stats, nil
}
