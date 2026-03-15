package models

import (
	"log/slog"
	"time"
)

type Quote struct {
	ID               int
	QuoteNumber      string
	Date             string
	SenderName       string
	SenderAddress    string
	RecipientName    string
	RecipientAddress string
	TaxRate          float64
	CreatedAt        time.Time
	Status           string // 'Entwurf', 'Verschickt', 'Angenommen', 'Abgelehnt', 'Umgewandelt'
	IsSmallBusiness  bool
	CustomerID       *int
	CustomerNumber   string
	Items            []QuoteItem
}

type QuoteItem struct {
	ID           int
	QuoteID      int
	Description  string
	Quantity     int
	PricePerUnit float64
	ProductID    *int
}

func (q *Quote) TotalNet() float64 {
	var total float64
	for _, item := range q.Items {
		total += float64(item.Quantity) * item.PricePerUnit
	}
	return total
}

func (q *Quote) TaxAmount() float64 {
	if q.IsSmallBusiness {
		return 0
	}
	return q.TotalNet() * (q.TaxRate / 100)
}

func (q *Quote) TotalGross() float64 {
	return q.TotalNet() + q.TaxAmount()
}

func (s *Store) CreateQuote(q *Quote) (int, error) {
	slog.Info("Creating quote", "quote_number", q.QuoteNumber, "customer_id", q.CustomerID)
	stx, err := s.Begin()
	if err != nil {
		slog.Error("Failed to begin transaction for quote creation", "error", err)
		return 0, err
	}
	tx := stx.Tx

	if q.Status == "" {
		q.Status = "Entwurf"
	}

	res, err := tx.Exec(`
		INSERT INTO quotes (quote_number, date, sender_name, sender_address, recipient_name, recipient_address, tax_rate, status, is_small_business, customer_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, q.QuoteNumber, q.Date, q.SenderName, q.SenderAddress, q.RecipientName, q.RecipientAddress, q.TaxRate, q.Status, q.IsSmallBusiness, q.CustomerID)
	if err != nil {
		slog.Error("Failed to insert quote", "quote_number", q.QuoteNumber, "error", err)
		tx.Rollback()
		return 0, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		slog.Error("Failed to get last insert id for quote", "error", err)
		tx.Rollback()
		return 0, err
	}

	for _, item := range q.Items {
		_, err := tx.Exec(`
			INSERT INTO quote_items (quote_id, description, quantity, price_per_unit, product_id)
			VALUES (?, ?, ?, ?, ?)
		`, id, item.Description, item.Quantity, item.PricePerUnit, item.ProductID)
		if err != nil {
			slog.Error("Failed to insert quote item", "quote_id", id, "description", item.Description, "error", err)
			tx.Rollback()
			return 0, err
		}
	}

	if err := tx.Commit(); err != nil {
		slog.Error("Failed to commit quote transaction", "quote_number", q.QuoteNumber, "error", err)
		return 0, err
	}

	slog.Info("Quote created successfully", "id", id, "quote_number", q.QuoteNumber)
	return int(id), nil
}

func (s *Store) ListQuotes() ([]Quote, error) {
	slog.Debug("Listing quotes from database")
	rows, err := s.DB.Query(`
		SELECT q.id, q.quote_number, q.date, q.recipient_name, q.status, q.tax_rate, q.is_small_business, q.customer_id, COALESCE(c.customer_number, '')
		FROM quotes q
		LEFT JOIN customers c ON q.customer_id = c.id
		ORDER BY q.id DESC
	`)
	if err != nil {
		slog.Error("Failed to query quotes", "error", err)
		return nil, err
	}
	defer rows.Close()

	var quotes []Quote
	for rows.Next() {
		var q Quote
		if err := rows.Scan(&q.ID, &q.QuoteNumber, &q.Date, &q.RecipientName, &q.Status, &q.TaxRate, &q.IsSmallBusiness, &q.CustomerID, &q.CustomerNumber); err != nil {
			slog.Error("Failed to scan quote row", "error", err)
			return nil, err
		}
		quotes = append(quotes, q)
	}
	return quotes, nil
}

func (s *Store) GetQuote(id int) (*Quote, error) {
	slog.Debug("Getting quote details", "id", id)
	var q Quote
	err := s.DB.QueryRow(`
		SELECT q.id, q.quote_number, q.date, q.sender_name, q.sender_address, q.recipient_name, q.recipient_address, q.tax_rate, q.created_at, q.status, q.is_small_business, q.customer_id, COALESCE(c.customer_number, '')
		FROM quotes q
		LEFT JOIN customers c ON q.customer_id = c.id
		WHERE q.id = ?
	`, id).Scan(&q.ID, &q.QuoteNumber, &q.Date, &q.SenderName, &q.SenderAddress, &q.RecipientName, &q.RecipientAddress, &q.TaxRate, &q.CreatedAt, &q.Status, &q.IsSmallBusiness, &q.CustomerID, &q.CustomerNumber)
	if err != nil {
		slog.Error("Failed to get quote", "id", id, "error", err)
		return nil, err
	}

	rows, err := s.DB.Query(`SELECT id, description, quantity, price_per_unit, product_id FROM quote_items WHERE quote_id = ?`, id)
	if err != nil {
		slog.Error("Failed to query quote items", "quote_id", id, "error", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item QuoteItem
		item.QuoteID = id
		if err := rows.Scan(&item.ID, &item.Description, &item.Quantity, &item.PricePerUnit, &item.ProductID); err != nil {
			slog.Error("Failed to scan quote item row", "quote_id", id, "error", err)
			return nil, err
		}
		q.Items = append(q.Items, item)
	}

	return &q, nil
}

func (s *Store) UpdateQuote(q *Quote) error {
	slog.Info("Updating quote", "id", q.ID, "quote_number", q.QuoteNumber)
	stx, err := s.Begin()
	if err != nil {
		slog.Error("Failed to begin transaction for quote update", "id", q.ID, "error", err)
		return err
	}
	tx := stx.Tx

	_, err = tx.Exec(`
		UPDATE quotes 
		SET quote_number = ?, date = ?, sender_name = ?, sender_address = ?, recipient_name = ?, recipient_address = ?, tax_rate = ?, status = ?, is_small_business = ?, customer_id = ?
		WHERE id = ?
	`, q.QuoteNumber, q.Date, q.SenderName, q.SenderAddress, q.RecipientName, q.RecipientAddress, q.TaxRate, q.Status, q.IsSmallBusiness, q.CustomerID, q.ID)
	if err != nil {
		slog.Error("Failed to update quote record", "id", q.ID, "error", err)
		tx.Rollback()
		return err
	}

	_, err = tx.Exec(`DELETE FROM quote_items WHERE quote_id = ?`, q.ID)
	if err != nil {
		slog.Error("Failed to delete old quote items", "id", q.ID, "error", err)
		tx.Rollback()
		return err
	}

	for _, item := range q.Items {
		_, err := tx.Exec(`
			INSERT INTO quote_items (quote_id, description, quantity, price_per_unit, product_id)
			VALUES (?, ?, ?, ?, ?)
		`, q.ID, item.Description, item.Quantity, item.PricePerUnit, item.ProductID)
		if err != nil {
			slog.Error("Failed to insert quote item during update", "id", q.ID, "description", item.Description, "error", err)
			tx.Rollback()
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		slog.Error("Failed to commit quote update transaction", "id", q.ID, "error", err)
		return err
	}
	slog.Info("Quote updated successfully", "id", q.ID)
	return nil
}

func (s *Store) DeleteQuote(id int) error {
	slog.Info("Deleting quote", "id", id)
	_, err := s.DB.Exec(`DELETE FROM quotes WHERE id = ?`, id)
	if err != nil {
		slog.Error("Failed to delete quote", "id", id, "error", err)
		return err
	}
	slog.Info("Quote deleted successfully", "id", id)
	return nil
}
