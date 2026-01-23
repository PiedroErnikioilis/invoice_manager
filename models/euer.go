package models

type EuerStats struct {
	TotalIncome   float64
	TotalExpenses float64
	Profit        float64
	Expenses      []Expense
	Invoices      []Invoice
}

func (s *Store) GetEuerStats() (*EuerStats, error) {
	stats := &EuerStats{}

	// 1. Calculate Income (Paid Invoices) and Load Invoices
	rows, err := s.DB.Query(`
		SELECT id, invoice_number, date, recipient_name, tax_rate, is_small_business, status
		FROM invoices
		WHERE status = 'Bezahlt'
		ORDER BY date DESC
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var i Invoice
			if err := rows.Scan(&i.ID, &i.InvoiceNumber, &i.Date, &i.RecipientName, &i.TaxRate, &i.IsSmallBusiness, &i.Status); err != nil {
				continue
			}

			// Load items for this invoice to calculate total
			fullInv, err := s.GetInvoice(i.ID)
			if err == nil {
				stats.TotalIncome += fullInv.TotalGross()
				stats.Invoices = append(stats.Invoices, *fullInv)
			}
		}
	}

	// 2. Calculate Expenses
	err = s.DB.QueryRow(`SELECT COALESCE(SUM(amount), 0) FROM expenses`).Scan(&stats.TotalExpenses)
	if err != nil {
		return nil, err
	}

	stats.Profit = stats.TotalIncome - stats.TotalExpenses
	
	// Load Expenses list for display
	stats.Expenses, _ = s.ListExpenses()

	return stats, nil
}
