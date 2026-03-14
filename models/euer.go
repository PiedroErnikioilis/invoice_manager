package models

import "fmt"

type EuerStats struct {
	Year          int
	TotalIncome   float64
	TotalExpenses float64
	Profit        float64
	Expenses      []Expense
	Invoices      []Invoice
}

// GetEuerStats returns income/expense statistics filtered by year.
// Year 0 means all years.
func (s *Store) GetEuerStats(year int) (*EuerStats, error) {
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

	rows, err := s.DB.Query(query)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var i Invoice
			if err := rows.Scan(&i.ID, &i.InvoiceNumber, &i.Date, &i.RecipientName, &i.TaxRate, &i.IsSmallBusiness, &i.Status); err != nil {
				continue
			}

			fullInv, err := s.GetInvoice(i.ID)
			if err == nil {
				stats.TotalIncome += fullInv.TotalGross()
				stats.Invoices = append(stats.Invoices, *fullInv)
			}
		}
	}

	// 2. Calculate Expenses
	expenseQuery := `SELECT COALESCE(SUM(amount), 0) FROM expenses`
	if year > 0 {
		expenseQuery += fmt.Sprintf(" WHERE date LIKE '%d-%%'", year)
	}
	err = s.DB.QueryRow(expenseQuery).Scan(&stats.TotalExpenses)
	if err != nil {
		return nil, err
	}

	stats.Profit = stats.TotalIncome - stats.TotalExpenses

	// Load Expenses list for display
	stats.Expenses, _ = s.ListExpenses(year)

	return stats, nil
}

// GetAvailableYears returns all years that have invoices or expenses.
func (s *Store) GetAvailableYears() ([]int, error) {
	rows, err := s.DB.Query(`
		SELECT DISTINCT year FROM (
			SELECT CAST(substr(date, 1, 4) AS INTEGER) AS year FROM invoices WHERE date != ''
			UNION
			SELECT CAST(substr(date, 1, 4) AS INTEGER) AS year FROM expenses WHERE date != ''
		) ORDER BY year DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var years []int
	for rows.Next() {
		var y int
		if err := rows.Scan(&y); err != nil {
			continue
		}
		if y > 0 {
			years = append(years, y)
		}
	}
	return years, nil
}
