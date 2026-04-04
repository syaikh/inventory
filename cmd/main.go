package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"inventory/internal/handlers"
	"inventory/internal/models"
	"inventory/internal/scanner"
	"inventory/internal/service"
	"inventory/internal/store"
)

// Configuration constants
const (
	defaultAddr     = ":8080"
	defaultScanMode = "in"
	defaultUnit     = "pcs"
)

// Config holds all application configuration
type Config struct {
	Addr        string
	DBDSN       string
	Device      string
	ScanMode    string
	ListDevices bool
}

func parseConfig() *Config {
	config := &Config{}

	flag.StringVar(&config.Addr, "addr", defaultAddr, "HTTP listen address")
	flag.StringVar(&config.DBDSN, "db", os.Getenv("DATABASE_URL"), "PostgreSQL DSN (default: env var DATABASE_URL)")
	flag.StringVar(&config.Device, "device", "", "USB HID event device, e.g. /dev/input/event3 (auto-detect if empty)")
	flag.StringVar(&config.ScanMode, "mode", defaultScanMode, "Default scan mode: 'in' or 'out'")
	flag.BoolVar(&config.ListDevices, "list-devices", false, "List detected HID input devices and exit")
	flag.Parse()

	return config
}

func initializeDatabase(dsn string) (*store.Store, error) {
	if dsn == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable not set and -db flag not provided")
	}

	s, err := store.New(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	log.Println("Database connected:", dsn)
	return s, nil
}

func setupScanner(device string) (*scanner.Scanner, error) {
	hidDevice := device
	if hidDevice == "" {
		devs, err := scanner.FindScanners()
		if err != nil {
			return nil, fmt.Errorf("failed to scan devices: %w", err)
		}
		if len(devs) > 0 {
			hidDevice = devs[0]
			log.Println("Auto-detected scanner device:", hidDevice)
		}
	}

	sc := scanner.New(hidDevice) // falls back to stdin if hidDevice == ""
	if hidDevice == "" {
		log.Println("No HID device — scanner reading from stdin (type barcode + Enter)")
	}

	return sc, nil
}

func startScanService(ctx context.Context, sc *scanner.Scanner, svc service.InventoryService, mode string) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Println("[SCAN] shutting down scanner service")
				return
			case ev := <-sc.Events():
				log.Printf("[SCAN] barcode=%s mode=%s", ev.Barcode, mode)
				scanEv := models.ScanEvent{
					Barcode: ev.Barcode,
					Mode:    mode,
					Qty:     1,
				}
				if _, err := svc.ProcessScan(scanEv); err != nil {
					log.Printf("[SCAN ERROR] %v", err)
				}
			case err := <-sc.Errors():
				log.Println("[SCANNER ERROR]", err)
			}
		}
	}()
}

func createHTTPServer(addr string, h *handlers.Handler) *http.Server {
	return &http.Server{
		Addr:         addr,
		Handler:      h.Routes(),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}
}

func main() {
	config := parseConfig()

	if config.ListDevices {
		devs, err := scanner.FindScanners()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error scanning devices:", err)
			os.Exit(1)
		}
		if len(devs) == 0 {
			fmt.Println("No HID keyboard/scanner devices found in /dev/input/")
		} else {
			fmt.Println("Detected devices:")
			for _, d := range devs {
				fmt.Println(" ", d)
			}
		}
		return
	}

	// Create a context that listens for the interrupt signal from the OS.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Initialize database
	dbStore, err := initializeDatabase(config.DBDSN)
	if err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}
	defer dbStore.Close()

	// Setup scanner
	hwScanner, err := setupScanner(config.Device)
	if err != nil {
		log.Fatalf("failed to setup scanner: %v", err)
	}

	// Validate scan mode
	mode := strings.ToLower(config.ScanMode)
	if mode != "in" && mode != "out" {
		mode = defaultScanMode
	}

	// Initialize Service Layer
	inventoryService := service.New(dbStore)

	// Initialize HTTP handler
	handler := handlers.New(inventoryService)

	// Start hardware scan service
	startScanService(ctx, hwScanner, inventoryService, mode)

	// Configure HTTP server
	srv := createHTTPServer(config.Addr, handler)

	// Start HTTP server in a goroutine
	go func() {
		log.Printf("Inventory server running at http://localhost%s", config.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	<-ctx.Done()
	log.Println("\nShutdown signal received, initiating graceful shutdown...")

	// Restore default behavior on the interrupt signal and notify user of shutdown.
	stop()

	// Perform graceful shutdown with a timeout (e.g., 5 seconds)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("HTTP server shutdown forced: %v", err)
	}

	log.Println("Server exited properly")
}
