package models

type Stats struct {
	TotalRevenueNet   float64
	TotalRevenueGross float64
	InvoicesCount     int
	DraftCount        int
	OpenCount         int
	PaidCount         int
	CancelledCount    int
	TopProducts       []TopProduct
}

type TopProduct struct {
	Name     string
	Quantity int
	Revenue  float64
}

func (s *Store) GetStats() (*Stats, error) {
	stats := &Stats{}

	// Revenue & Count
	err := s.DB.QueryRow(`
		SELECT
			COUNT(*),
			COALESCE(SUM(CASE WHEN status = 'Entwurf' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'Offen' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'Bezahlt' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'Storniert' THEN 1 ELSE 0 END), 0)
		FROM invoices
	`).Scan(&stats.InvoicesCount, &stats.DraftCount, &stats.OpenCount, &stats.PaidCount, &stats.CancelledCount)
	if err != nil {
		return nil, err
	}

	// Calculate Revenue (More complex due to structure, lets iterate or use advanced SQL)
	// Simple SQL for Gross/Net is hard because tax is per invoice and items are separate.
	// But we can approximate or join.
	// Let's do a join.
	// Net = Sum(quantity * price)
	// Gross = Net + Tax (if not small business)

	rows, err := s.DB.Query(`
		SELECT i.tax_rate, i.is_small_business, ii.quantity, ii.price_per_unit
		FROM invoices i
		JOIN invoice_items ii ON i.id = ii.invoice_id
		WHERE i.status NOT IN ('Entwurf', 'Storniert')
	`)
	if err != nil {
		return nil, err
	}
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
		stats.TotalRevenueNet += net

		if !isSmall {
			stats.TotalRevenueGross += net * (1 + taxRate/100)
		} else {
			stats.TotalRevenueGross += net
		}
	}

	// Top Products
	pRows, err := s.DB.Query(`
		SELECT p.name, SUM(ii.quantity) as qty, SUM(ii.quantity * ii.price_per_unit) as rev
		FROM invoice_items ii
		JOIN products p ON ii.product_id = p.id
		JOIN invoices i ON ii.invoice_id = i.id
		WHERE i.status NOT IN ('Entwurf', 'Storniert')
		GROUP BY p.id
		ORDER BY rev DESC
		LIMIT 5
	`)
	if err == nil {
		defer pRows.Close()
		for pRows.Next() {
			var tp TopProduct
			if err := pRows.Scan(&tp.Name, &tp.Quantity, &tp.Revenue); err == nil {
				stats.TopProducts = append(stats.TopProducts, tp)
			}
		}
	}

	return stats, nil
}
