package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

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

func startScanService(sc *scanner.Scanner, svc service.InventoryService, mode string) {
	go func() {
		for {
			select {
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

func startHTTPServer(addr string, h *handlers.Handler) error {
	log.Printf("Inventory server running at http://localhost%s", addr)
	return http.ListenAndServe(addr, h.Routes())
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

	// Initialize database
	store, err := initializeDatabase(config.DBDSN)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()

	// Setup scanner
	hwScanner, err := setupScanner(config.Device)
	if err != nil {
		log.Fatal(err)
	}

	// Validate scan mode
	mode := strings.ToLower(config.ScanMode)
	if mode != "in" && mode != "out" {
		mode = defaultScanMode
	}

	// Initialize Service Layer
	inventoryService := service.New(store)

	// Initialize HTTP handler
	handler := handlers.New(inventoryService)

	// Start hardware scan service to translate hardware events to Service events
	startScanService(hwScanner, inventoryService, mode)

	// Start HTTP server
	if err := startHTTPServer(config.Addr, handler); err != nil {
		log.Fatal("HTTP server error:", err)
	}
}
