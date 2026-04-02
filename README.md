# Inventory System

A lightweight inventory management system written in Go with USB barcode scanner support.

## Features

- Real-time USB barcode scanner integration (HID keyboard emulation)
- Web UI: dashboard, item manager, scanner page, transaction history
- SQLite database (no external DB required)
- Server-Sent Events for live scan feed in browser
- Auto-create new items on first scan
- Scan IN / Scan OUT modes
- REST API

---

## Requirements

- Go 1.21+
- GCC (for go-sqlite3 CGO): `sudo apt install gcc` or `brew install gcc`
- Linux: read permission on `/dev/input/eventX` for raw HID mode

---

## Build & Run

```bash
# Install dependency
go get github.com/mattn/go-sqlite3

# Build
go build -o inventory ./cmd/

# Run (auto-detects USB scanner)
./inventory

# Or specify device explicitly
./inventory -device /dev/input/event3

# List detected HID devices
./inventory -list-devices

# Default scan mode (in or out)
./inventory -mode in

# Custom port and DB path
./inventory -addr :9000 -db /var/data/inventory.db
```

Open **http://localhost:8080** in your browser.

---

## USB Scanner Setup (Linux)

Most USB barcode scanners register as HID keyboard devices. The app handles them in two ways:

### 1. Raw HID Mode (recommended — works without focus)

The scanner is read directly from `/dev/input/eventX`. This means scans are captured even when the browser window is not focused.

```bash
# Find your device
./inventory -list-devices
# Example output:
#   /dev/input/event3  (Honeywell Barcode Scanner)

# Run with explicit device
./inventory -device /dev/input/event3
```

If you get a permission error:
```bash
# Add yourself to the input group
sudo usermod -aG input $USER
# Or set a udev rule (more permanent):
echo 'SUBSYSTEM=="input", GROUP="input", MODE="0660"' | sudo tee /etc/udev/rules.d/99-input.rules
sudo udevadm control --reload-rules
```

### 2. Stdin / TTY Mode (fallback)

If no device is specified or detected, the app reads from stdin. The OS translates the HID input to keypresses:

```bash
./inventory   # reads barcodes from stdin
```

In this mode, make sure the terminal running the server has focus when scanning, OR pipe from a dedicated input reader.

---

## API Reference

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/scan` | Process a scan `{barcode, mode, qty}` |
| GET | `/api/items` | List items `?search=&category=` |
| POST | `/api/items` | Create/update item |
| GET | `/api/items/:id` | Get item by ID |
| PUT | `/api/items/:id` | Update item |
| DELETE | `/api/items/:id` | Delete item |
| GET | `/api/transactions` | Transaction history `?barcode=&limit=` |
| GET | `/api/categories` | List categories |
| GET | `/api/stats` | Dashboard stats |
| GET | `/api/events` | SSE stream for live scans |

### Scan example

```bash
curl -X POST http://localhost:8080/api/scan \
  -H 'Content-Type: application/json' \
  -d '{"barcode":"012345678905","mode":"in","qty":1}'
```

---

## Project Structure

```
inventory/
├── cmd/
│   └── main.go              # Entry point, wires scanner → store → HTTP
├── internal/
│   ├── models/models.go     # Item, Transaction, ScanEvent types
│   ├── scanner/scanner.go   # USB HID + stdin barcode reader
│   ├── store/store.go       # SQLite persistence layer
│   └── handlers/handlers.go # HTTP API + SSE
├── web/static/
│   └── index.html           # Single-page web UI
├── go.mod
└── README.md
```

---

## Windows / macOS Notes

On **Windows**, USB scanners in HID mode type into whichever window has focus. Run the server in the background and keep the browser window focused, or use the `-mode stdin` approach with a dedicated input program.

On **macOS**, raw `/dev/input` does not exist. Use stdin mode or read from the IOHIDManager using CGO bindings (not included).
