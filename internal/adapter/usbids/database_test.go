package usbids

import (
	"strings"
	"testing"
)

func TestParseAndLookup(t *testing.T) {
	t.Parallel()

	db := Parse(strings.NewReader(`# comment
0000  Zero Vendor
	0001  Zero Product
0001  First Vendor
	0001  First Product
		0001  Interface Name
0002  Second Vendor
	0003  Second Product
C 00  Unclassified
	01  Device
`))

	tests := []struct {
		name        string
		idVendor    uint16
		idProduct   uint16
		wantVendor  string
		wantProduct string
	}{
		{
			name:        "zero vendor",
			idVendor:    0x0000,
			idProduct:   0x0001,
			wantVendor:  "Zero Vendor",
			wantProduct: "Zero Product",
		},
		{
			name:        "vendor and product",
			idVendor:    0x0001,
			idProduct:   0x0001,
			wantVendor:  "First Vendor",
			wantProduct: "First Product",
		},
		{
			name:        "known vendor unknown product",
			idVendor:    0x0002,
			idProduct:   0xffff,
			wantVendor:  "Second Vendor",
			wantProduct: "",
		},
		{
			name:        "unknown vendor",
			idVendor:    0xffff,
			idProduct:   0xffff,
			wantVendor:  "",
			wantProduct: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			vendorName, productName := db.Lookup(tt.idVendor, tt.idProduct)
			if vendorName != tt.wantVendor || productName != tt.wantProduct {
				t.Fatalf(
					"Lookup(%04x, %04x)=(%q, %q), want (%q, %q)",
					tt.idVendor,
					tt.idProduct,
					vendorName,
					productName,
					tt.wantVendor,
					tt.wantProduct,
				)
			}
		})
	}
}

func TestParseIgnoresMalformedEntries(t *testing.T) {
	t.Parallel()

	db := Parse(strings.NewReader(`invalid vendor
	0001  Orphan Product
0001
0002  Valid Vendor
	invalid product
	0003  Valid Product
`))

	vendorName, productName := db.Lookup(0x0002, 0x0003)
	if vendorName != "Valid Vendor" || productName != "Valid Product" {
		t.Fatalf("Lookup=(%q, %q), want valid names", vendorName, productName)
	}
}
