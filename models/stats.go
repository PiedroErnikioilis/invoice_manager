package models

import "log/slog"

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
	slog.Debug("Executing GetStats")
	stats := &Stats{}

	// Revenue & Count
	slog.Debug("Querying invoice status counts")
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
		slog.Error("Failed to query stats", "error", err)
		return nil, err
	}
	slog.Debug("Invoice counts retrieved", "total", stats.InvoicesCount, "paid", stats.PaidCount)

	slog.Debug("Querying revenue data items")
	rows, err := s.DB.Query(`
		SELECT i.tax_rate, i.is_small_business, ii.quantity, ii.price_per_unit
		FROM invoices i
		JOIN invoice_items ii ON i.id = ii.invoice_id
		WHERE i.status NOT IN ('Entwurf', 'Storniert')
	`)
	if err != nil {
		slog.Error("Failed to query revenue rows", "error", err)
		return nil, err
	}
	defer rows.Close()

	itemCount := 0
	for rows.Next() {
		var taxRate float64
		var isSmall bool
		var qty int
		var price float64
		if err := rows.Scan(&taxRate, &isSmall, &qty, &price); err != nil {
			slog.Error("Failed to scan revenue row", "error", err)
			continue
		}

		net := float64(qty) * price
		stats.TotalRevenueNet += net

		if !isSmall {
			stats.TotalRevenueGross += net * (1 + taxRate/100)
		} else {
			stats.TotalRevenueGross += net
		}
		itemCount++
	}
	slog.Debug("Revenue calculation complete", "items_processed", itemCount, "total_net", stats.TotalRevenueNet)

	// Top Products
	slog.Debug("Querying top products")
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
	if err != nil {
		slog.Error("Failed to query top products", "error", err)
	} else {
		defer pRows.Close()
		topCount := 0
		for pRows.Next() {
			var tp TopProduct
			if err := pRows.Scan(&tp.Name, &tp.Quantity, &tp.Revenue); err == nil {
				stats.TopProducts = append(stats.TopProducts, tp)
				topCount++
			} else {
				slog.Error("Failed to scan top product row", "error", err)
			}
		}
		slog.Debug("Top products retrieved", "count", topCount)
	}

	slog.Info("Stats generation complete")
	return stats, nil
}
