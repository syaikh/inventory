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
	"inventory/internal/store"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	dbDSN := flag.String("db", os.Getenv("DATABASE_URL"), "PostgreSQL DSN (default: env var DATABASE_URL)")
	device := flag.String("device", "", "USB HID event device, e.g. /dev/input/event3 (auto-detect if empty)")
	scanMode := flag.String("mode", "in", "Default scan mode: 'in' or 'out'")
	listDevices := flag.Bool("list-devices", false, "List detected HID input devices and exit")
	flag.Parse()

	if *dbDSN == "" {
		log.Fatal("DATABASE_URL environment variable not set and -db flag not provided")
	}

	if *listDevices {
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

	// Open DB
	s, err := store.New(*dbDSN)
	if err != nil {
		log.Fatal("DB error:", err)
	}
	defer s.Close()
	log.Println("Database connected:", *dbDSN)

	// Resolve HID device
	hidDevice := *device
	if hidDevice == "" {
		devs, _ := scanner.FindScanners()
		if len(devs) > 0 {
			hidDevice = devs[0]
			log.Println("Auto-detected scanner device:", hidDevice)
		}
	}

	// Start scanner
	mode := strings.ToLower(*scanMode)
	if mode != "in" && mode != "out" {
		mode = "in"
	}

	sc := scanner.New(hidDevice) // falls back to stdin if hidDevice == ""
	if hidDevice == "" {
		log.Println("No HID device — scanner reading from stdin (type barcode + Enter)")
	}

	// HTTP handler
	h := handlers.New(s)

	// Bridge scanner events → HTTP handler scan bus
	go func() {
		for {
			select {
			case ev := <-sc.Events():
				log.Printf("[SCAN] barcode=%s mode=%s", ev.Barcode, mode)
				// Treat as qty=1 in default mode
				h.ScanBus() <- models.ScanEvent{Barcode: ev.Barcode, Mode: mode, Qty: 1}
				// Also persist via store
				item, _ := s.GetItem(ev.Barcode)
				if item == nil {
					item = &models.Item{Barcode: ev.Barcode, Name: "Unknown — " + ev.Barcode, Unit: "pcs"}
					s.UpsertItem(item)
					item, _ = s.GetItem(ev.Barcode)
				}
				delta := 1
				txType := "scan_in"
				if mode == "out" {
					delta = -1
					txType = "scan_out"
				}
				s.UpdateQuantity(ev.Barcode, delta)
				s.AddTransaction(&models.Transaction{
					Barcode:  ev.Barcode,
					ItemName: item.Name,
					Type:     txType,
					Quantity: 1,
				})

			case err := <-sc.Errors():
				log.Println("[SCANNER ERROR]", err)
			}
		}
	}()

	log.Printf("Inventory server running at http://localhost%s", *addr)
	log.Fatal(http.ListenAndServe(*addr, h.Routes()))
}
