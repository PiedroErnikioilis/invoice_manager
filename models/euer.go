package models

type EuerStats struct {
	TotalIncome   float64
	TotalExpenses float64
	Profit        float64
	Expenses      []Expense
}

func (s *Store) GetEuerStats() (*EuerStats, error) {
	stats := &EuerStats{}

	// 1. Calculate Income (Paid Invoices)
	// We assume Gross amount is the income (Brutto).
	
	rows, err := s.DB.Query(`
		SELECT i.tax_rate, i.is_small_business, ii.quantity, ii.price_per_unit
		FROM invoices i
		JOIN invoice_items ii ON i.id = ii.invoice_id
		WHERE i.status = 'Bezahlt'
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var taxRate float64
			var isSmall bool
			var qty int
			var price float64
			if err := rows.Scan(&taxRate, &isSmall, &qty, &price); err != nil {
				continue
			}
			
			net := float64(qty) * price
			if !isSmall {
				stats.TotalIncome += net * (1 + taxRate/100)
			} else {
				stats.TotalIncome += net
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
