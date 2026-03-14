package models

import (
	"time"
)

type CreditNote struct {
	ID               int
	CreditNoteNumber string
	Date             string
	SenderName       string
	SenderAddress    string
	RecipientName    string
	RecipientAddress string
	TaxRate          float64
	CreatedAt        time.Time
	Status           string // 'Entwurf', 'Offen', 'Abgeschlossen'
	IsSmallBusiness  bool
	CustomerID       *int
	InvoiceID        *int // Reference to original invoice
	Items            []CreditNoteItem
}

type CreditNoteItem struct {
	ID           int
	CreditNoteID int
	Description  string
	Quantity     int
	PricePerUnit float64
	ProductID    *int
}

// TotalNet returns the negative net total (since it is a credit)
func (c *CreditNote) TotalNet() float64 {
	var total float64
	for _, item := range c.Items {
		total += float64(item.Quantity) * item.PricePerUnit
	}
	return -total
}

func (c *CreditNote) TaxAmount() float64 {
	if c.IsSmallBusiness {
		return 0
	}
	return c.TotalNet() * (c.TaxRate / 100)
}

func (c *CreditNote) TotalGross() float64 {
	return c.TotalNet() + c.TaxAmount()
}

func (s *Store) CreateCreditNote(c *CreditNote) (int, error) {
	stx, err := s.Begin()
	if err != nil {
		return 0, err
	}
	tx := stx.Tx

	if c.Status == "" {
		c.Status = "Offen"
	}

	res, err := tx.Exec(`
		INSERT INTO credit_notes (credit_note_number, date, sender_name, sender_address, recipient_name, recipient_address, tax_rate, status, is_small_business, customer_id, invoice_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, c.CreditNoteNumber, c.Date, c.SenderName, c.SenderAddress, c.RecipientName, c.RecipientAddress, c.TaxRate, c.Status, c.IsSmallBusiness, c.CustomerID, c.InvoiceID)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	for _, item := range c.Items {
		_, err := tx.Exec(`
			INSERT INTO credit_note_items (credit_note_id, description, quantity, price_per_unit, product_id)
			VALUES (?, ?, ?, ?, ?)
		`, id, item.Description, item.Quantity, item.PricePerUnit, item.ProductID)
		if err != nil {
			tx.Rollback()
			return 0, err
		}

		if item.ProductID != nil {
			// Record stock movement: Credit Note means IN (+quantity) if goods are returned
			// We assume standard credit note returns goods.
			err := s.RecordStockMovementTx(stx, *item.ProductID, item.Quantity, "CANCELLATION", "Gutschrift "+c.CreditNoteNumber)
			if err != nil {
				tx.Rollback()
				return 0, err
			}
		}
	}

	return int(id), tx.Commit()
}

func (s *Store) ListCreditNotes() ([]CreditNote, error) {
	rows, err := s.DB.Query(`SELECT id, credit_note_number, date, recipient_name, status, tax_rate, is_small_business FROM credit_notes ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []CreditNote
	for rows.Next() {
		var c CreditNote
		if err := rows.Scan(&c.ID, &c.CreditNoteNumber, &c.Date, &c.RecipientName, &c.Status, &c.TaxRate, &c.IsSmallBusiness); err != nil {
			return nil, err
		}
		notes = append(notes, c)
	}
	return notes, nil
}

func (s *Store) GetCreditNote(id int) (*CreditNote, error) {
	var c CreditNote
	err := s.DB.QueryRow(`
		SELECT id, credit_note_number, date, sender_name, sender_address, recipient_name, recipient_address, tax_rate, created_at, status, is_small_business, customer_id, invoice_id
		FROM credit_notes WHERE id = ?
	`, id).Scan(&c.ID, &c.CreditNoteNumber, &c.Date, &c.SenderName, &c.SenderAddress, &c.RecipientName, &c.RecipientAddress, &c.TaxRate, &c.CreatedAt, &c.Status, &c.IsSmallBusiness, &c.CustomerID, &c.InvoiceID)
	if err != nil {
		return nil, err
	}

	rows, err := s.DB.Query(`SELECT id, description, quantity, price_per_unit, product_id FROM credit_note_items WHERE credit_note_id = ?`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item CreditNoteItem
		item.CreditNoteID = id
		if err := rows.Scan(&item.ID, &item.Description, &item.Quantity, &item.PricePerUnit, &item.ProductID); err != nil {
			return nil, err
		}
		c.Items = append(c.Items, item)
	}

	return &c, nil
}

func (s *Store) DeleteCreditNote(id int) error {
	_, err := s.DB.Exec(`DELETE FROM credit_notes WHERE id = ?`, id)
	return err
}
