package handlers

import (
	"strconv"
	"strings"
)

// parseDecimal parst einen Dezimalwert und akzeptiert sowohl Punkt als auch Komma
// als Dezimaltrennzeichen (z.B. "1234,56" oder "1234.56").
// Tausendertrennzeichen (Punkt vor Komma) werden ebenfalls korrekt behandelt:
// "1.234,56" → 1234.56
func parseDecimal(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}

	// Wenn sowohl Punkt als auch Komma vorhanden sind,
	// ist das letzte Trennzeichen der Dezimaltrenner.
	lastDot := strings.LastIndex(s, ".")
	lastComma := strings.LastIndex(s, ",")

	if lastComma > lastDot {
		// Deutsches Format: 1.234,56 → Punkte sind Tausendertrenner, Komma ist Dezimal
		s = strings.ReplaceAll(s, ".", "")
		s = strings.Replace(s, ",", ".", 1)
	} else if lastDot > lastComma {
		// Englisches Format: 1,234.56 → Kommas sind Tausendertrenner
		s = strings.ReplaceAll(s, ",", "")
	} else if lastComma >= 0 {
		// Nur Komma vorhanden: 123,45
		s = strings.Replace(s, ",", ".", 1)
	}
	// Nur Punkt oder kein Trennzeichen → bereits korrekt

	v, _ := strconv.ParseFloat(s, 64)
	return v
}
