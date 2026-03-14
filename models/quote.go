package models

import (
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
	stx, err := s.Begin()
	if err != nil {
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
		tx.Rollback()
		return 0, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	for _, item := range q.Items {
		_, err := tx.Exec(`
			INSERT INTO quote_items (quote_id, description, quantity, price_per_unit, product_id)
			VALUES (?, ?, ?, ?, ?)
		`, id, item.Description, item.Quantity, item.PricePerUnit, item.ProductID)
		if err != nil {
			tx.Rollback()
			return 0, err
		}
	}

	return int(id), tx.Commit()
}

func (s *Store) ListQuotes() ([]Quote, error) {
	rows, err := s.DB.Query(`SELECT id, quote_number, date, recipient_name, status, tax_rate, is_small_business FROM quotes ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var quotes []Quote
	for rows.Next() {
		var q Quote
		if err := rows.Scan(&q.ID, &q.QuoteNumber, &q.Date, &q.RecipientName, &q.Status, &q.TaxRate, &q.IsSmallBusiness); err != nil {
			return nil, err
		}
		quotes = append(quotes, q)
	}
	return quotes, nil
}

func (s *Store) GetQuote(id int) (*Quote, error) {
	var q Quote
	err := s.DB.QueryRow(`
		SELECT id, quote_number, date, sender_name, sender_address, recipient_name, recipient_address, tax_rate, created_at, status, is_small_business, customer_id
		FROM quotes WHERE id = ?
	`, id).Scan(&q.ID, &q.QuoteNumber, &q.Date, &q.SenderName, &q.SenderAddress, &q.RecipientName, &q.RecipientAddress, &q.TaxRate, &q.CreatedAt, &q.Status, &q.IsSmallBusiness, &q.CustomerID)
	if err != nil {
		return nil, err
	}

	rows, err := s.DB.Query(`SELECT id, description, quantity, price_per_unit, product_id FROM quote_items WHERE quote_id = ?`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item QuoteItem
		item.QuoteID = id
		if err := rows.Scan(&item.ID, &item.Description, &item.Quantity, &item.PricePerUnit, &item.ProductID); err != nil {
			return nil, err
		}
		q.Items = append(q.Items, item)
	}

	return &q, nil
}

func (s *Store) UpdateQuote(q *Quote) error {
	stx, err := s.Begin()
	if err != nil {
		return err
	}
	tx := stx.Tx

	_, err = tx.Exec(`
		UPDATE quotes 
		SET quote_number = ?, date = ?, sender_name = ?, sender_address = ?, recipient_name = ?, recipient_address = ?, tax_rate = ?, status = ?, is_small_business = ?, customer_id = ?
		WHERE id = ?
	`, q.QuoteNumber, q.Date, q.SenderName, q.SenderAddress, q.RecipientName, q.RecipientAddress, q.TaxRate, q.Status, q.IsSmallBusiness, q.CustomerID, q.ID)
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec(`DELETE FROM quote_items WHERE quote_id = ?`, q.ID)
	if err != nil {
		tx.Rollback()
		return err
	}

	for _, item := range q.Items {
		_, err := tx.Exec(`
			INSERT INTO quote_items (quote_id, description, quantity, price_per_unit, product_id)
			VALUES (?, ?, ?, ?, ?)
		`, q.ID, item.Description, item.Quantity, item.PricePerUnit, item.ProductID)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

func (s *Store) DeleteQuote(id int) error {
	_, err := s.DB.Exec(`DELETE FROM quotes WHERE id = ?`, id)
	return err
}
