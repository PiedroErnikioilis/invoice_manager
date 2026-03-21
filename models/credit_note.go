package models

import (
	"log/slog"
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
	CustomerNumber   string
	InvoiceID        *int // Reference to original invoice
	InternalNote     string
	DocumentNote     string
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
	slog.Debug("Executing CreateCreditNote", "credit_note_number", c.CreditNoteNumber, "customer_id", c.CustomerID)
	stx, err := s.Begin()
	if err != nil {
		slog.Error("Failed to begin transaction for credit note creation", "error", err)
		return 0, err
	}
	tx := stx.Tx

	if c.Status == "" {
		c.Status = "Offen"
	}

	slog.Debug("Inserting credit note into database", "credit_note_number", c.CreditNoteNumber)
	res, err := tx.Exec(`
		INSERT INTO credit_notes (credit_note_number, date, sender_name, sender_address, recipient_name, recipient_address, tax_rate, status, is_small_business, customer_id, invoice_id, internal_note, document_note)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, c.CreditNoteNumber, c.Date, c.SenderName, c.SenderAddress, c.RecipientName, c.RecipientAddress, c.TaxRate, c.Status, c.IsSmallBusiness, c.CustomerID, c.InvoiceID, c.InternalNote, c.DocumentNote)
	if err != nil {
		slog.Error("Failed to insert credit note", "credit_note_number", c.CreditNoteNumber, "error", err)
		tx.Rollback()
		return 0, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		slog.Error("Failed to get last insert id for credit note", "error", err)
		tx.Rollback()
		return 0, err
	}
	slog.Debug("Credit note record inserted successfully", "id", id)

	for _, item := range c.Items {
		slog.Debug("Inserting credit note item", "credit_note_id", id, "description", item.Description)
		_, err := tx.Exec(`
			INSERT INTO credit_note_items (credit_note_id, description, quantity, price_per_unit, product_id)
			VALUES (?, ?, ?, ?, ?)
		`, id, item.Description, item.Quantity, item.PricePerUnit, item.ProductID)
		if err != nil {
			slog.Error("Failed to insert credit note item", "credit_note_id", id, "description", item.Description, "error", err)
			tx.Rollback()
			return 0, err
		}

		if item.ProductID != nil {
			// Record stock movement: Credit Note means IN (+quantity) if goods are returned
			// We assume standard credit note returns goods.
			slog.Debug("Recording stock return for credit note", "product_id", *item.ProductID, "quantity", item.Quantity)
			err := s.RecordStockMovementTx(stx, *item.ProductID, item.Quantity, "CANCELLATION", "Gutschrift "+c.CreditNoteNumber)
			if err != nil {
				slog.Error("Failed to record stock movement for credit note", "product_id", *item.ProductID, "error", err)
				tx.Rollback()
				return 0, err
			}
		}
	}

	if err := tx.Commit(); err != nil {
		slog.Error("Failed to commit credit note transaction", "credit_note_number", c.CreditNoteNumber, "error", err)
		return 0, err
	}

	slog.Info("CreateCreditNote completed successfully", "id", id, "credit_note_number", c.CreditNoteNumber)
	return int(id), nil
}

func (s *Store) ListCreditNotes() ([]CreditNote, error) {
	slog.Debug("Executing ListCreditNotes")
	query := `
		SELECT cn.id, cn.credit_note_number, cn.date, cn.recipient_name, cn.status, cn.tax_rate, cn.is_small_business, cn.customer_id, COALESCE(c.customer_number, '')
		FROM credit_notes cn
		LEFT JOIN customers c ON cn.customer_id = c.id
		ORDER BY cn.id DESC
	`
	slog.Debug("Querying credit notes from database", "query", query)
	rows, err := s.DB.Query(query)
	if err != nil {
		slog.Error("Failed to query credit notes", "error", err)
		return nil, err
	}
	defer rows.Close()

	var notes []CreditNote
	for rows.Next() {
		var c CreditNote
		if err := rows.Scan(&c.ID, &c.CreditNoteNumber, &c.Date, &c.RecipientName, &c.Status, &c.TaxRate, &c.IsSmallBusiness, &c.CustomerID, &c.CustomerNumber); err != nil {
			slog.Error("Failed to scan credit note row", "error", err)
			return nil, err
		}
		notes = append(notes, c)
	}
	slog.Info("ListCreditNotes completed successfully", "count", len(notes))
	return notes, nil
}

func (s *Store) GetCreditNote(id int) (*CreditNote, error) {
	slog.Debug("Executing GetCreditNote", "id", id)
	var c CreditNote
	query := `
		SELECT cn.id, cn.credit_note_number, cn.date, cn.sender_name, cn.sender_address, cn.recipient_name, cn.recipient_address, cn.tax_rate, cn.created_at, cn.status, cn.is_small_business, cn.customer_id, COALESCE(cust.customer_number, ''), cn.invoice_id, COALESCE(cn.internal_note, ''), COALESCE(cn.document_note, '')
		FROM credit_notes cn
		LEFT JOIN customers cust ON cn.customer_id = cust.id
		WHERE cn.id = ?
	`
	slog.Debug("Querying credit note details", "id", id, "query", query)
	err := s.DB.QueryRow(query, id).Scan(&c.ID, &c.CreditNoteNumber, &c.Date, &c.SenderName, &c.SenderAddress, &c.RecipientName, &c.RecipientAddress, &c.TaxRate, &c.CreatedAt, &c.Status, &c.IsSmallBusiness, &c.CustomerID, &c.CustomerNumber, &c.InvoiceID, &c.InternalNote, &c.DocumentNote)
	if err != nil {
		slog.Error("Failed to get credit note", "id", id, "error", err)
		return nil, err
	}

	itemQuery := `SELECT id, description, quantity, price_per_unit, product_id FROM credit_note_items WHERE credit_note_id = ?`
	slog.Debug("Querying credit note items", "credit_note_id", id, "query", itemQuery)
	rows, err := s.DB.Query(itemQuery, id)
	if err != nil {
		slog.Error("Failed to query credit note items", "credit_note_id", id, "error", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item CreditNoteItem
		item.CreditNoteID = id
		if err := rows.Scan(&item.ID, &item.Description, &item.Quantity, &item.PricePerUnit, &item.ProductID); err != nil {
			slog.Error("Failed to scan credit note item row", "credit_note_id", id, "error", err)
			return nil, err
		}
		c.Items = append(c.Items, item)
	}

	slog.Info("GetCreditNote completed successfully", "id", id, "items_count", len(c.Items))
	return &c, nil
}

func (s *Store) DeleteCreditNote(id int) error {
	slog.Debug("Executing DeleteCreditNote", "id", id)
	_, err := s.DB.Exec(`DELETE FROM credit_notes WHERE id = ?`, id)
	if err != nil {
		slog.Error("Failed to delete credit note", "id", id, "error", err)
		return err
	}
	slog.Info("DeleteCreditNote completed successfully", "id", id)
	return nil
}
