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
	GetItem(barcode string) (*models.Item, error)
	GetItemByID(id int64) (*models.Item, error)
	UpsertItem(item *models.Item) error
	UpdateQuantity(barcode string, delta int) (*models.Item, error)
	DeleteItem(id int64) error
	ListItems(search, category string) ([]models.Item, error)
	ListTransactions(barcode string, limit int) ([]models.Transaction, error)
	Categories() ([]string, error)
	Stats() (map[string]any, error)
	AddTransaction(tx *models.Transaction) error
}

// InventoryService defines the business logic contract used by handlers.
type InventoryService interface {
	// Business operations
	ProcessScan(ev models.ScanEvent) (*models.Item, error)
	ScanBus() <-chan models.ScanEvent

	// CRUD operations
	GetItem(barcode string) (*models.Item, error)
	GetItemByID(id int64) (*models.Item, error)
	UpsertItem(item *models.Item) error
	DeleteItem(id int64) error
	ListItems(search, category string) ([]models.Item, error)
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
func (s *Service) ProcessScan(ev models.ScanEvent) (*models.Item, error) {
	// Broadcast the event so UI can react immediately
	s.broadcastScanEvent(ev)

	// Get or create item
	item, err := s.getOrCreateItem(ev.Barcode)
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
	updatedItem, err := s.store.UpdateQuantity(ev.Barcode, delta)
	if err != nil {
		return nil, fmt.Errorf("failed to update quantity for %s: %w", ev.Barcode, err)
	}

	// Record transaction
	transaction := &models.Transaction{
		Barcode:  ev.Barcode,
		ItemName: item.Name,
		Type:     txType,
		Quantity: ev.Qty, 
	}

	if err := s.store.AddTransaction(transaction); err != nil {
		return nil, fmt.Errorf("failed to add transaction for %s: %w", ev.Barcode, err)
	}

	return updatedItem, nil
}

// getOrCreateItem retrieves an item or creates an "Unknown" one if it doesn't exist
func (s *Service) getOrCreateItem(barcode string) (*models.Item, error) {
	item, err := s.store.GetItem(barcode)
	if err != nil {
		return nil, fmt.Errorf("failed to get item %s: %w", barcode, err)
	}

	if item == nil {
		// Create unknown item
		item = &models.Item{
			Barcode: barcode,
			Name:    "Unknown — " + barcode,
			Unit:    "pcs",
		}
		if err := s.store.UpsertItem(item); err != nil {
			return nil, fmt.Errorf("failed to create unknown item %s: %w", barcode, err)
		}
		// Retrieve the created item with ID
		item, err = s.store.GetItem(barcode)
		if err != nil {
			return nil, fmt.Errorf("failed to get created item %s: %w", barcode, err)
		}
	}

	return item, nil
}

// GetItem retrieves an item by its barcode
func (s *Service) GetItem(barcode string) (*models.Item, error) {
	return s.store.GetItem(barcode)
}

// GetItemByID retrieves an item by its ID
func (s *Service) GetItemByID(id int64) (*models.Item, error) {
	return s.store.GetItemByID(id)
}

// UpsertItem creates or updates an item
func (s *Service) UpsertItem(item *models.Item) error {
	return s.store.UpsertItem(item)
}

// DeleteItem removes an item by its ID
func (s *Service) DeleteItem(id int64) error {
	return s.store.DeleteItem(id)
}

// ListItems returns a list of items based on search and category filters
func (s *Service) ListItems(search, category string) ([]models.Item, error) {
	return s.store.ListItems(search, category)
}

// ListTransactions returns a list of transactions for a barcode, with an optional limit
func (s *Service) ListTransactions(barcode string, limit int) ([]models.Transaction, error) {
	return s.store.ListTransactions(barcode, limit)
}

// Categories returns all unique item categories
func (s *Service) Categories() ([]string, error) {
	return s.store.Categories()
}

// Stats returns aggregation statistics across the inventory
func (s *Service) Stats() (map[string]any, error) {
	return s.store.Stats()
}
