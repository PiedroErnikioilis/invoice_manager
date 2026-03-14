package models

import (
	"fmt"
	"strings"
)

// FormatCurrency formats a float64 as a German-style currency string (e.g., "1.234,56 €").
func FormatCurrency(amount float64) string {
	negative := amount < 0
	if negative {
		amount = -amount
	}

	// Format with 2 decimal places
	s := fmt.Sprintf("%.2f", amount)

	// Split into integer and decimal parts
	parts := strings.Split(s, ".")
	intPart := parts[0]
	decPart := parts[1]

	// Add thousands separator (dot)
	var result []byte
	for i, d := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			result = append(result, '.')
		}
		result = append(result, byte(d))
	}

	formatted := string(result) + "," + decPart + " €"
	if negative {
		formatted = "-" + formatted
	}
	return formatted
}
