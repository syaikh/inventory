// Package scanner reads from a USB barcode scanner.
// Most USB barcode scanners work as HID keyboard devices — they "type"
// the barcode followed by Enter. This package supports two modes:
//
//  1. HID raw device mode (Linux /dev/input/eventX) — reads keycodes
//     directly without needing the scanner to be the focused window.
//  2. Stdin mode — for systems where the OS already translates the HID
//     device to a TTY, or for testing via keyboard / pipe.
package scanner

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Event is emitted whenever a complete barcode is scanned.
type Event struct {
	Barcode string
	At      time.Time
}

// Scanner reads barcodes from a source.
type Scanner struct {
	events chan Event
	errors chan error
	done   chan struct{}
}

// New creates a Scanner. If devicePath is empty it falls back to stdin.
// devicePath should be a Linux event device like /dev/input/event0,
// or a path returned by FindScanners().
func New(devicePath string) *Scanner {
	s := &Scanner{
		events: make(chan Event, 32),
		errors: make(chan error, 8),
		done:   make(chan struct{}),
	}
	if devicePath != "" {
		go s.readHID(devicePath)
	} else {
		go s.readStdin()
	}
	return s
}

// Events returns the channel of scan events.
func (s *Scanner) Events() <-chan Event { return s.events }

// Errors returns the channel of read errors.
func (s *Scanner) Errors() <-chan error { return s.errors }

// Close stops the scanner.
func (s *Scanner) Close() { close(s.done) }

// FindScanners returns candidate /dev/input/eventX paths that look like
// keyboard/HID devices (heuristic: check for "kbd" or "barcode" in name).
func FindScanners() ([]string, error) {
	entries, err := filepath.Glob("/sys/class/input/event*/device/name")
	if err != nil {
		return nil, err
	}
	var found []string
	for _, p := range entries {
		data, _ := os.ReadFile(p)
		name := strings.ToLower(strings.TrimSpace(string(data)))
		if strings.Contains(name, "barcode") ||
			strings.Contains(name, "scanner") ||
			strings.Contains(name, "keyboard") ||
			strings.Contains(name, "kbd") {
			// derive /dev/input/eventX from /sys/class/input/eventX/device/name
			base := filepath.Base(filepath.Dir(filepath.Dir(p)))
			found = append(found, "/dev/input/"+base)
		}
	}
	return found, nil
}

// ---- stdin mode ----

func (s *Scanner) readStdin() {
	sc := bufio.NewScanner(os.Stdin)
	for {
		select {
		case <-s.done:
			return
		default:
		}
		if sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line != "" {
				s.events <- Event{Barcode: line, At: time.Now()}
			}
		} else {
			if err := sc.Err(); err != nil {
				s.errors <- err
			}
			return
		}
	}
}

// ---- HID raw mode ----
// Linux input_event struct: timeval (8 bytes), type (2), code (2), value (4) = 24 bytes total.
// EV_KEY=1, value=1 means key down. We translate key codes to ASCII chars.

type inputEvent struct {
	Sec   uint64 // timeval seconds
	Usec  uint64 // timeval microseconds (or nanoseconds on 64-bit)
	Type  uint16
	Code  uint16
	Value int32
}

const evKey = 1

// keyMap maps Linux key codes to (normal, shifted) rune pairs.
var keyMap = map[uint16][2]rune{
	2: {'1', '!'}, 3: {'2', '@'}, 4: {'3', '#'}, 5: {'4', '$'},
	6: {'5', '%'}, 7: {'6', '^'}, 8: {'7', '&'}, 9: {'8', '*'},
	10: {'9', '('}, 11: {'0', ')'}, 12: {'-', '_'}, 13: {'=', '+'},
	16: {'q', 'Q'}, 17: {'w', 'W'}, 18: {'e', 'E'}, 19: {'r', 'R'},
	20: {'t', 'T'}, 21: {'y', 'Y'}, 22: {'u', 'U'}, 23: {'i', 'I'},
	24: {'o', 'O'}, 25: {'p', 'P'}, 26: {'[', '{'}, 27: {']', '}'},
	30: {'a', 'A'}, 31: {'s', 'S'}, 32: {'d', 'D'}, 33: {'f', 'F'},
	34: {'g', 'G'}, 35: {'h', 'H'}, 36: {'j', 'J'}, 37: {'k', 'K'},
	38: {'l', 'L'}, 39: {';', ':'}, 40: {'\'', '"'}, 43: {'\\', '|'},
	44: {'z', 'Z'}, 45: {'x', 'X'}, 46: {'c', 'C'}, 47: {'v', 'V'},
	48: {'b', 'B'}, 49: {'n', 'N'}, 50: {'m', 'M'}, 51: {',', '<'},
	52: {'.', '>'}, 53: {'/', '?'},
}

const keyEnter = 28
const keyLeftShift = 42
const keyRightShift = 54

func (s *Scanner) readHID(path string) {
	f, err := os.Open(path)
	if err != nil {
		s.errors <- fmt.Errorf("open %s: %w", path, err)
		return
	}
	defer f.Close()

	var buf strings.Builder
	var shift bool
	var ev inputEvent

	for {
		select {
		case <-s.done:
			return
		default:
		}

		if err := binary.Read(f, binary.LittleEndian, &ev); err != nil {
			if err == io.EOF {
				return
			}
			s.errors <- err
			return
		}

		if ev.Type != evKey {
			continue
		}

		// Track shift state
		if ev.Code == keyLeftShift || ev.Code == keyRightShift {
			shift = ev.Value == 1 || ev.Value == 2
			continue
		}

		// Key down or repeat
		if ev.Value != 1 {
			continue
		}

		if ev.Code == keyEnter {
			barcode := strings.TrimSpace(buf.String())
			if barcode != "" {
				s.events <- Event{Barcode: barcode, At: time.Now()}
			}
			buf.Reset()
			continue
		}

		if pair, ok := keyMap[ev.Code]; ok {
			if shift {
				buf.WriteRune(pair[1])
			} else {
				buf.WriteRune(pair[0])
			}
		}
	}
}
