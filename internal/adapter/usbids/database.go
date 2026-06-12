// Package usbids resolves USB vendor and product IDs using the Linux usb.ids database.
package usbids

import (
	"bufio"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
)

const (
	minEntryFields = 2

	hwdataPath   = "/usr/share/hwdata/usb.ids"
	miscPath     = "/usr/share/misc/usb.ids"
	sharePath    = "/usr/share/usb.ids"
	usbutilsPath = "/var/lib/usbutils/usb.ids"
)

type vendor struct {
	name     string
	products map[uint16]string
}

// Database contains USB vendor and product names.
type Database struct {
	vendors map[uint16]vendor
}

// Parse reads a usb.ids database.
func Parse(reader io.Reader) Database {
	db := Database{
		vendors: make(map[uint16]vendor),
	}

	var (
		currentVendor    uint16
		hasCurrentVendor bool
	)

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "C ") {
			hasCurrentVendor = false

			continue
		}

		if strings.HasPrefix(line, "\t\t") {
			continue
		}

		productLine, isProduct := strings.CutPrefix(line, "\t")
		if isProduct {
			if hasCurrentVendor {
				db.addProduct(currentVendor, productLine)
			}

			continue
		}

		id, name, ok := parseEntry(line)
		if !ok {
			hasCurrentVendor = false

			continue
		}

		currentVendor = id
		hasCurrentVendor = true
		db.vendors[id] = vendor{
			name:     name,
			products: make(map[uint16]string),
		}
	}

	return db
}

// LoadDefault loads the first available standard Linux usb.ids database.
func LoadDefault() Database {
	paths := [...]string{hwdataPath, miscPath, sharePath, usbutilsPath}
	for _, path := range paths {
		file, err := os.Open(path) //nolint:gosec // Paths are fixed system usb.ids locations.
		if err != nil {
			continue
		}

		db := Parse(file)
		_ = file.Close()

		return db
	}

	return Database{
		vendors: make(map[uint16]vendor),
	}
}

// NewDefaultLookup returns a lazy lookup backed by the first available standard Linux usb.ids database.
func NewDefaultLookup() func(idVendor, idProduct uint16) (string, string) {
	var (
		db   Database
		once sync.Once
	)

	return func(idVendor, idProduct uint16) (string, string) {
		once.Do(func() {
			db = LoadDefault()
		})

		return db.Lookup(idVendor, idProduct)
	}
}

// Lookup returns vendor and product names. Missing names are returned as empty strings.
func (db Database) Lookup(idVendor, idProduct uint16) (string, string) {
	item, ok := db.vendors[idVendor]
	if !ok {
		return "", ""
	}

	return item.name, item.products[idProduct]
}

func (db Database) addProduct(idVendor uint16, line string) {
	item, ok := db.vendors[idVendor]
	if !ok {
		return
	}

	idProduct, name, ok := parseEntry(line)
	if !ok {
		return
	}

	item.products[idProduct] = name
	db.vendors[idVendor] = item
}

func parseEntry(line string) (uint16, string, bool) {
	fields := strings.Fields(line)
	if len(fields) < minEntryFields {
		return 0, "", false
	}

	id, err := strconv.ParseUint(fields[0], 16, 16)
	if err != nil || len(fields[0]) != 4 {
		return 0, "", false
	}

	name := strings.TrimSpace(strings.TrimPrefix(line, fields[0]))
	if name == "" {
		return 0, "", false
	}

	return uint16(id), name, true
}
