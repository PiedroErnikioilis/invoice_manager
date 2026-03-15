package models

import (
	"log/slog"
	"time"
)

type Invoice struct {
	ID               int
	InvoiceNumber    string
	Date             string
	SenderName       string
	SenderAddress    string
	RecipientName    string
	RecipientAddress string
	TaxRate          float64
	CreatedAt        time.Time
	Status           string // 'Entwurf', 'Offen', 'Bezahlt'
	IsSmallBusiness  bool
	CustomerID       *int
	CustomerNumber   string
	Items            []InvoiceItem
	ItemCount        int // nur für Listenansicht (via Subquery)
}

type InvoiceItem struct {
	ID           int
	InvoiceID    int
	Description  string
	Quantity     int
	PricePerUnit float64
	ProductID    *int // Nullable
}

func (i *Invoice) TotalNet() float64 {
	var total float64
	for _, item := range i.Items {
		total += float64(item.Quantity) * item.PricePerUnit
	}
	return total
}

func (i *Invoice) TaxAmount() float64 {
	if i.IsSmallBusiness {
		return 0
	}
	return i.TotalNet() * (i.TaxRate / 100)
}

func (i *Invoice) TotalGross() float64 {
	return i.TotalNet() + i.TaxAmount()
}

func (s *Store) CreateInvoice(inv *Invoice) (int, error) {
	slog.Info("Creating invoice", "invoice_number", inv.InvoiceNumber, "customer_id", inv.CustomerID)
	stx, err := s.Begin()
	if err != nil {
		slog.Error("Failed to begin transaction for invoice creation", "error", err)
		return 0, err
	}
	tx := stx.Tx

	// Default status if empty
	if inv.Status == "" {
		inv.Status = "Entwurf"
	}

	res, err := tx.Exec(`
		INSERT INTO invoices (invoice_number, date, sender_name, sender_address, recipient_name, recipient_address, tax_rate, status, is_small_business, customer_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, inv.InvoiceNumber, inv.Date, inv.SenderName, inv.SenderAddress, inv.RecipientName, inv.RecipientAddress, inv.TaxRate, inv.Status, inv.IsSmallBusiness, inv.CustomerID)
	if err != nil {
		slog.Error("Failed to insert invoice", "invoice_number", inv.InvoiceNumber, "error", err)
		tx.Rollback()
		return 0, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		slog.Error("Failed to get last insert id for invoice", "error", err)
		tx.Rollback()
		return 0, err
	}

	for _, item := range inv.Items {
		_, err := tx.Exec(`
			INSERT INTO invoice_items (invoice_id, description, quantity, price_per_unit, product_id)
			VALUES (?, ?, ?, ?, ?)
		`, id, item.Description, item.Quantity, item.PricePerUnit, item.ProductID)
		if err != nil {
			slog.Error("Failed to insert invoice item", "invoice_id", id, "description", item.Description, "error", err)
			tx.Rollback()
			return 0, err
		}

		if item.ProductID != nil {
			// Record stock movement: Invoice means OUT (-quantity)
			slog.Debug("Recording stock deduction for invoice", "product_id", *item.ProductID, "quantity", item.Quantity)
			err := s.RecordStockMovementTx(stx, *item.ProductID, -item.Quantity, "INVOICE", "Rechnung "+inv.InvoiceNumber)
			if err != nil {
				slog.Error("Failed to record stock movement for invoice", "product_id", *item.ProductID, "error", err)
				tx.Rollback()
				return 0, err
			}
		}
	}

	if err := tx.Commit(); err != nil {
		slog.Error("Failed to commit invoice transaction", "invoice_number", inv.InvoiceNumber, "error", err)
		return 0, err
	}

	slog.Info("Invoice created successfully", "id", id, "invoice_number", inv.InvoiceNumber)
	return int(id), nil
}

func (s *Store) UpdateInvoice(inv *Invoice) error {
	slog.Info("Updating invoice", "id", inv.ID, "invoice_number", inv.InvoiceNumber)
	stx, err := s.Begin()
	if err != nil {
		slog.Error("Failed to begin transaction for invoice update", "id", inv.ID, "error", err)
		return err
	}
	tx := stx.Tx

	_, err = tx.Exec(`
		UPDATE invoices 
		SET invoice_number = ?, date = ?, sender_name = ?, sender_address = ?, recipient_name = ?, recipient_address = ?, tax_rate = ?, status = ?, is_small_business = ?, customer_id = ?
		WHERE id = ?
	`, inv.InvoiceNumber, inv.Date, inv.SenderName, inv.SenderAddress, inv.RecipientName, inv.RecipientAddress, inv.TaxRate, inv.Status, inv.IsSmallBusiness, inv.CustomerID, inv.ID)
	if err != nil {
		slog.Error("Failed to update invoice record", "id", inv.ID, "error", err)
		tx.Rollback()
		return err
	}

	// Restore stock for items being deleted (Cancel previous Invoice booking)
	slog.Debug("Restoring stock for old invoice items before update", "id", inv.ID)
	rows, err := tx.Query(`SELECT product_id, quantity FROM invoice_items WHERE invoice_id = ?`, inv.ID)
	if err != nil {
		slog.Error("Failed to query old invoice items for stock restoration", "id", inv.ID, "error", err)
		tx.Rollback()
		return err
	}

	type itemToRestore struct {
		ProductID *int
		Quantity  int
	}
	var toRestore []itemToRestore

	for rows.Next() {
		var itr itemToRestore
		if err := rows.Scan(&itr.ProductID, &itr.Quantity); err != nil {
			rows.Close()
			tx.Rollback()
			return err
		}
		toRestore = append(toRestore, itr)
	}
	rows.Close()

	for _, itr := range toRestore {
		if itr.ProductID != nil {
			// Restore: Positive quantity
			err := s.RecordStockMovementTx(stx, *itr.ProductID, itr.Quantity, "INVOICE_UPDATE", "Korrektur Rechnung "+inv.InvoiceNumber)
			if err != nil {
				slog.Error("Failed to restore stock for old invoice item", "product_id", *itr.ProductID, "error", err)
				tx.Rollback()
				return err
			}
		}
	}

	// Delete existing items
	_, err = tx.Exec(`DELETE FROM invoice_items WHERE invoice_id = ?`, inv.ID)
	if err != nil {
		slog.Error("Failed to delete old invoice items", "id", inv.ID, "error", err)
		tx.Rollback()
		return err
	}

	// Insert new items and deduct stock
	for _, item := range inv.Items {
		_, err := tx.Exec(`
			INSERT INTO invoice_items (invoice_id, description, quantity, price_per_unit, product_id)
			VALUES (?, ?, ?, ?, ?)
		`, inv.ID, item.Description, item.Quantity, item.PricePerUnit, item.ProductID)
		if err != nil {
			slog.Error("Failed to insert new invoice item during update", "id", inv.ID, "description", item.Description, "error", err)
			tx.Rollback()
			return err
		}

		if item.ProductID != nil {
			// Deduct again
			slog.Debug("Recording stock deduction for new invoice item during update", "product_id", *item.ProductID, "quantity", item.Quantity)
			err := s.RecordStockMovementTx(stx, *item.ProductID, -item.Quantity, "INVOICE", "Rechnung "+inv.InvoiceNumber)
			if err != nil {
				slog.Error("Failed to record stock movement for new invoice item", "product_id", *item.ProductID, "error", err)
				tx.Rollback()
				return err
			}
		}
	}

	if err := tx.Commit(); err != nil {
		slog.Error("Failed to commit invoice update transaction", "id", inv.ID, "error", err)
		return err
	}
	slog.Info("Invoice updated successfully", "id", inv.ID)
	return nil
}

func (s *Store) CancelInvoice(id int) error {
	slog.Info("Cancelling invoice", "id", id)
	stx, err := s.Begin()
	if err != nil {
		slog.Error("Failed to begin transaction for invoice cancellation", "id", id, "error", err)
		return err
	}
	tx := stx.Tx

	// Get invoice number for movement note
	var invoiceNumber string
	err = tx.QueryRow(`SELECT invoice_number FROM invoices WHERE id = ?`, id).Scan(&invoiceNumber)
	if err != nil {
		slog.Error("Failed to get invoice number for cancellation", "id", id, "error", err)
		tx.Rollback()
		return err
	}

	// Restore stock for all items with a product_id
	slog.Debug("Restoring stock for cancelled invoice", "id", id, "invoice_number", invoiceNumber)
	rows, err := tx.Query(`SELECT product_id, quantity FROM invoice_items WHERE invoice_id = ? AND product_id IS NOT NULL`, id)
	if err != nil {
		slog.Error("Failed to query invoice items for cancellation", "id", id, "error", err)
		tx.Rollback()
		return err
	}

	type itemToRestore struct {
		ProductID int
		Quantity  int
	}
	var toRestore []itemToRestore

	for rows.Next() {
		var itr itemToRestore
		if err := rows.Scan(&itr.ProductID, &itr.Quantity); err != nil {
			rows.Close()
			tx.Rollback()
			return err
		}
		toRestore = append(toRestore, itr)
	}
	rows.Close()

	for _, itr := range toRestore {
		err := s.RecordStockMovementTx(stx, itr.ProductID, itr.Quantity, "CANCELLATION", "Storno Rechnung "+invoiceNumber)
		if err != nil {
			slog.Error("Failed to restore stock for cancelled invoice item", "product_id", itr.ProductID, "error", err)
			tx.Rollback()
			return err
		}
	}

	// Set status to cancelled
	_, err = tx.Exec(`UPDATE invoices SET status = 'Storniert' WHERE id = ?`, id)
	if err != nil {
		slog.Error("Failed to update invoice status to cancelled", "id", id, "error", err)
		tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		slog.Error("Failed to commit invoice cancellation transaction", "id", id, "error", err)
		return err
	}
	slog.Info("Invoice cancelled successfully", "id", id, "invoice_number", invoiceNumber)
	return nil
}

// InvoiceFilter enthält Such- und Sortierparameter für die Rechnungsliste.
type InvoiceFilter struct {
	Search string // Freitext-Suche (Nr, Empfänger, Kunden-ID)
	Status string // Filter nach Status (leer = alle)
	Sort   string // Spalte: date, number, recipient, status, items
	Order  string // asc oder desc
}

// AllowedSort gibt die SQL-Spalte für den Sort-Parameter zurück.
func (f InvoiceFilter) OrderByClause() string {
	col := "i.id"
	switch f.Sort {
	case "number":
		col = "i.invoice_number"
	case "date":
		col = "i.date"
	case "recipient":
		col = "i.recipient_name"
	case "status":
		col = "i.status"
	case "items":
		col = "item_count"
	}
	dir := "DESC"
	if f.Order == "asc" {
		dir = "ASC"
	}
	return col + " " + dir
}

func (s *Store) ListInvoices(filter ...InvoiceFilter) ([]Invoice, error) {
	slog.Debug("Listing invoices from database")
	query := `
		SELECT i.id, i.invoice_number, i.date, i.sender_name, i.recipient_name,
		       i.tax_rate, i.status, i.is_small_business, i.customer_id, COALESCE(c.customer_number, ''),
		       (SELECT COUNT(*) FROM invoice_items ii WHERE ii.invoice_id = i.id) AS item_count
		FROM invoices i
		LEFT JOIN customers c ON i.customer_id = c.id`

	var args []interface{}
	var conditions []string

	var f InvoiceFilter
	if len(filter) > 0 {
		f = filter[0]
	}

	if f.Search != "" {
		conditions = append(conditions,
			"(i.invoice_number LIKE ? OR i.recipient_name LIKE ? OR c.customer_number LIKE ?)")
		like := "%" + f.Search + "%"
		args = append(args, like, like, like)
	}
	if f.Status != "" {
		conditions = append(conditions, "i.status = ?")
		args = append(args, f.Status)
	}

	if len(conditions) > 0 {
		query += " WHERE " + conditions[0]
		for _, c := range conditions[1:] {
			query += " AND " + c
		}
	}

	query += " ORDER BY " + f.OrderByClause()

	rows, err := s.DB.Query(query, args...)
	if err != nil {
		slog.Error("Failed to query invoices", "error", err)
		return nil, err
	}
	defer rows.Close()

	var invoices []Invoice
	for rows.Next() {
		var i Invoice
		if err := rows.Scan(&i.ID, &i.InvoiceNumber, &i.Date, &i.SenderName, &i.RecipientName, &i.TaxRate, &i.Status, &i.IsSmallBusiness, &i.CustomerID, &i.CustomerNumber, &i.ItemCount); err != nil {
			slog.Error("Failed to scan invoice row", "error", err)
			return nil, err
		}
		invoices = append(invoices, i)
	}
	return invoices, nil
}

func (s *Store) GetInvoice(id int) (*Invoice, error) {
	slog.Debug("Getting invoice details", "id", id)
	var i Invoice
	err := s.DB.QueryRow(`
		SELECT i.id, i.invoice_number, i.date, i.sender_name, i.sender_address, i.recipient_name, i.recipient_address, i.tax_rate, i.created_at, i.status, i.is_small_business, i.customer_id, COALESCE(c.customer_number, '')
		FROM invoices i
		LEFT JOIN customers c ON i.customer_id = c.id
		WHERE i.id = ?
	`, id).Scan(&i.ID, &i.InvoiceNumber, &i.Date, &i.SenderName, &i.SenderAddress, &i.RecipientName, &i.RecipientAddress, &i.TaxRate, &i.CreatedAt, &i.Status, &i.IsSmallBusiness, &i.CustomerID, &i.CustomerNumber)
	if err != nil {
		slog.Error("Failed to get invoice", "id", id, "error", err)
		return nil, err
	}

	rows, err := s.DB.Query(`SELECT id, description, quantity, price_per_unit, product_id FROM invoice_items WHERE invoice_id = ?`, id)
	if err != nil {
		slog.Error("Failed to query invoice items", "invoice_id", id, "error", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item InvoiceItem
		item.InvoiceID = id
		if err := rows.Scan(&item.ID, &item.Description, &item.Quantity, &item.PricePerUnit, &item.ProductID); err != nil {
			slog.Error("Failed to scan invoice item row", "invoice_id", id, "error", err)
			return nil, err
		}
		i.Items = append(i.Items, item)
	}

	return &i, nil
}
