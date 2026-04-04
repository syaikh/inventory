package service

import (
	"fmt"
	"inventory/internal/models"
)

const (
	scanBusBufferSize = 32
)

// StoreInterface defines the contract with the data layer.
type StoreInterface interface {
	GetProduct(barcode string) (*models.Product, error)
	GetProductByID(id int64) (*models.Product, error)
	UpsertProduct(product *models.Product) error
	UpdateQuantity(barcode string, delta int) (*models.Product, error)
	DeleteProduct(id int64) error
	ListProducts(search, category string) ([]models.Product, error)
	ListTransactions(barcode string, limit int) ([]models.Transaction, error)
	Categories() ([]string, error)
	Stats() (map[string]any, error)
	AddTransaction(tx *models.Transaction) error
}

// InventoryService defines the business logic contract used by handlers.
type InventoryService interface {
	// Business operations
	ProcessScan(ev models.ScanEvent) (*models.Product, error)
	ScanBus() <-chan models.ScanEvent

	// CRUD operations
	GetProduct(barcode string) (*models.Product, error)
	GetProductByID(id int64) (*models.Product, error)
	UpsertProduct(product *models.Product) error
	DeleteProduct(id int64) error
	ListProducts(search, category string) ([]models.Product, error)
	ListTransactions(barcode string, limit int) ([]models.Transaction, error)
	Categories() ([]string, error)
	Stats() (map[string]any, error)
}

// Service is the concrete implementation of InventoryService.
type Service struct {
	store   StoreInterface
	scanBus chan models.ScanEvent
}

// New creates a new Service instance.
func New(store StoreInterface) *Service {
	return &Service{
		store:   store,
		scanBus: make(chan models.ScanEvent, scanBusBufferSize),
	}
}

// ScanBus returns a read-only channel for Server-Sent Events to consume.
func (s *Service) ScanBus() <-chan models.ScanEvent {
	return s.scanBus
}

func (s *Service) broadcastScanEvent(ev models.ScanEvent) {
	select {
	case s.scanBus <- ev:
	default:
		// Channel is full, drop event to avoid blocking
	}
}

// ProcessScan handles the business logic of updating items and creating transactions based on a scan string.
func (s *Service) ProcessScan(ev models.ScanEvent) (*models.Product, error) {
	// Broadcast the event so UI can react immediately
	s.broadcastScanEvent(ev)

	// Get or create product
	product, err := s.getOrCreateProduct(ev.Barcode)
	if err != nil {
		return nil, err
	}

	// Calculate quantity change
	delta := ev.Qty
	if delta == 0 {
		delta = 1
	}

	txType := "scan_in"
	if ev.Mode == "out" {
		delta = -delta
		txType = "scan_out"
	}

	// Update quantity
	updatedProduct, err := s.store.UpdateQuantity(ev.Barcode, delta)
	if err != nil {
		return nil, fmt.Errorf("failed to update quantity for %s: %w", ev.Barcode, err)
	}

	// Record transaction
	transaction := &models.Transaction{
		Barcode:     ev.Barcode,
		ProductName: product.Name,
		Type:        txType,
		Quantity:    ev.Qty, 
	}

	if err := s.store.AddTransaction(transaction); err != nil {
		return nil, fmt.Errorf("failed to add transaction for %s: %w", ev.Barcode, err)
	}

	return updatedProduct, nil
}

// getOrCreateProduct retrieves a product or creates an "Unknown" one if it doesn't exist
func (s *Service) getOrCreateProduct(barcode string) (*models.Product, error) {
	product, err := s.store.GetProduct(barcode)
	if err != nil {
		return nil, fmt.Errorf("failed to get product %s: %w", barcode, err)
	}

	if product == nil {
		// Create unknown product
		product = &models.Product{
			Barcodes: []string{barcode},
			Name:     "Unknown — " + barcode,
			Unit:     "pcs",
		}
		if err := s.store.UpsertProduct(product); err != nil {
			return nil, fmt.Errorf("failed to create unknown product %s: %w", barcode, err)
		}
		// Retrieve the created product with ID
		product, err = s.store.GetProduct(barcode)
		if err != nil {
			return nil, fmt.Errorf("failed to get created product %s: %w", barcode, err)
		}
	}

	return product, nil
}

// GetProduct retrieves a product by its primary barcode search
func (s *Service) GetProduct(barcode string) (*models.Product, error) {
	return s.store.GetProduct(barcode)
}

// GetProductByID retrieves a product by its ID
func (s *Service) GetProductByID(id int64) (*models.Product, error) {
	return s.store.GetProductByID(id)
}

// UpsertProduct creates or updates a product
func (s *Service) UpsertProduct(product *models.Product) error {
	return s.store.UpsertProduct(product)
}

// DeleteProduct removes a product by its ID
func (s *Service) DeleteProduct(id int64) error {
	return s.store.DeleteProduct(id)
}

// ListProducts returns a list of products based on search and category filters
func (s *Service) ListProducts(search, category string) ([]models.Product, error) {
	return s.store.ListProducts(search, category)
}

// ListTransactions returns a list of transactions for a barcode, with an optional limit
func (s *Service) ListTransactions(barcode string, limit int) ([]models.Transaction, error) {
	return s.store.ListTransactions(barcode, limit)
}

// Categories returns all unique product categories
func (s *Service) Categories() ([]string, error) {
	return s.store.Categories()
}

// Stats returns aggregation statistics across the inventory
func (s *Service) Stats() (map[string]any, error) {
	return s.store.Stats()
}

