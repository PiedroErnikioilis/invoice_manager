package models

import (
	"fmt"
	"log/slog"
)

type CategoryStat struct {
	Name       string
	Total      float64
	Percentage float64
}

type EuerStats struct {
	Year          int
	TotalIncomeNet   float64
	TotalIncomeVat   float64
	TotalIncomeGross float64
	TotalExpensesNet   float64
	TotalExpensesTax   float64
	TotalExpensesGross float64
	Profit        float64
	VatPayable    float64 // Zahllast (Income VAT - Expense Tax)
	Expenses      []Expense
	Invoices      []Invoice
	CategoryStats []CategoryStat
}

// GetEuerStats returns income/expense statistics filtered by year.
func (s *Store) GetEuerStats(year int) (*EuerStats, error) {
	slog.Debug("Executing GetEuerStats", "year", year)
	stats := &EuerStats{Year: year}

	// 1. Calculate Income (Paid Invoices)
	var dateFilter string
	if year > 0 {
		dateFilter = fmt.Sprintf(" AND date LIKE '%d-%%'", year)
	}

	query := fmt.Sprintf(`
		SELECT id, invoice_number, date, recipient_name, tax_rate, is_small_business, status
		FROM invoices
		WHERE status = 'Bezahlt'%s
		ORDER BY date DESC
	`, dateFilter)

	slog.Debug("Querying paid invoices for EÜR", "year", year, "query", query)
	rows, err := s.DB.Query(query)
	if err != nil {
		slog.Error("Failed to query paid invoices for EÜR", "year", year, "error", err)
	} else {
		defer rows.Close()
		invoiceCount := 0
		for rows.Next() {
			invoiceCount++
			var i Invoice
			if err := rows.Scan(&i.ID, &i.InvoiceNumber, &i.Date, &i.RecipientName, &i.TaxRate, &i.IsSmallBusiness, &i.Status); err != nil {
				slog.Error("Failed to scan invoice row for EÜR", "error", err)
				continue
			}

			slog.Debug("Fetching full invoice details for EÜR", "id", i.ID, "invoice_number", i.InvoiceNumber)
			fullInv, err := s.GetInvoice(i.ID)
			if err != nil {
				slog.Error("Failed to get full invoice details for EÜR", "id", i.ID, "error", err)
				continue
			}
			stats.TotalIncomeNet += fullInv.TotalNet()
			stats.TotalIncomeVat += fullInv.TaxAmount()
			stats.TotalIncomeGross += fullInv.TotalGross()
			stats.Invoices = append(stats.Invoices, *fullInv)
		}
		slog.Debug("Finished parsing paid invoices for EÜR", "year", year, "count", invoiceCount)
	}

	// 1b. Calculate Credits (Credit Notes)
	creditQuery := fmt.Sprintf(`
		SELECT id, credit_note_number, date, recipient_name, tax_rate, is_small_business, status
		FROM credit_notes
		WHERE status != 'Entwurf'%s
		ORDER BY date DESC
	`, dateFilter)

	slog.Debug("Querying credit notes for EÜR", "year", year, "query", creditQuery)
	cRows, err := s.DB.Query(creditQuery)
	if err != nil {
		slog.Error("Failed to query credit notes for EÜR", "year", year, "error", err)
	} else {
		defer cRows.Close()
		creditCount := 0
		for cRows.Next() {
			creditCount++
			var cn CreditNote
			if err := cRows.Scan(&cn.ID, &cn.CreditNoteNumber, &cn.Date, &cn.RecipientName, &cn.TaxRate, &cn.IsSmallBusiness, &cn.Status); err != nil {
				slog.Error("Failed to scan credit note row for EÜR", "error", err)
				continue
			}

			slog.Debug("Fetching full credit note details for EÜR", "id", cn.ID, "credit_note_number", cn.CreditNoteNumber)
			fullCn, err := s.GetCreditNote(cn.ID)
			if err != nil {
				slog.Error("Failed to get full credit note details for EÜR", "id", cn.ID, "error", err)
				continue
			}
			stats.TotalIncomeNet += fullCn.TotalNet()
			stats.TotalIncomeVat += fullCn.TaxAmount()
			stats.TotalIncomeGross += fullCn.TotalGross()
		}
		slog.Debug("Finished parsing credit notes for EÜR", "year", year, "count", creditCount)
	}

	// 2. Load Expenses list
	slog.Debug("Listing expenses for EÜR", "year", year)
	expenses, err := s.ListExpenses(year)
	if err != nil {
		slog.Error("Failed to list expenses for EÜR", "year", year, "error", err)
	}
	stats.Expenses = expenses
	for _, e := range stats.Expenses {
		stats.TotalExpensesNet += e.Net()
		stats.TotalExpensesTax += e.Tax()
		stats.TotalExpensesGross += e.Amount
	}
	slog.Debug("Finished calculating expense totals for EÜR", "year", year, "count", len(expenses))

	// 3. Category Stats (using Net)
	slog.Debug("Calculating category stats for EÜR", "year", year)
	catSums := make(map[string]float64)
	for _, e := range stats.Expenses {
		catName := e.CategoryName
		if catName == "" {
			catName = "Sonstige"
		}
		catSums[catName] += e.Net()
	}

	for name, sum := range catSums {
		cs := CategoryStat{Name: name, Total: sum}
		if stats.TotalExpensesNet > 0 {
			cs.Percentage = (cs.Total / stats.TotalExpensesNet) * 100
		}
		stats.CategoryStats = append(stats.CategoryStats, cs)
	}

	stats.Profit = stats.TotalIncomeNet - stats.TotalExpensesNet
	stats.VatPayable = stats.TotalIncomeVat - stats.TotalExpensesTax

	slog.Info("GetEuerStats completed successfully", "year", year, "profit", stats.Profit)
	return stats, nil
}

// GetAvailableYears returns all years that have invoices or expenses.
func (s *Store) GetAvailableYears() ([]int, error) {
	slog.Debug("Executing GetAvailableYears")
	query := `
		SELECT DISTINCT year FROM (
			SELECT CAST(substr(date, 1, 4) AS INTEGER) AS year FROM invoices WHERE date != ''
			UNION
			SELECT CAST(substr(date, 1, 4) AS INTEGER) AS year FROM expenses WHERE date != ''
            UNION
            SELECT CAST(substr(date, 1, 4) AS INTEGER) AS year FROM credit_notes WHERE date != ''
		) ORDER BY year DESC
	`
	slog.Debug("Querying available years", "query", query)
	rows, err := s.DB.Query(query)
	if err != nil {
		slog.Error("Failed to query available years", "error", err)
		return nil, err
	}
	defer rows.Close()

	var years []int
	for rows.Next() {
		var y int
		if err := rows.Scan(&y); err != nil {
			slog.Error("Failed to scan year row", "error", err)
			continue
		}
		if y > 0 {
			years = append(years, y)
		}
	}
	slog.Info("GetAvailableYears completed successfully", "count", len(years))
	return years, nil
}
