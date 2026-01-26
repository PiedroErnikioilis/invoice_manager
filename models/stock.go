package models

import (
	"time"
)

type StockMovement struct {
	ID           int
	ProductID    int
	Quantity     int
	MovementType string
	Note         string
	CreatedAt    time.Time
}

func (s *Store) RecordStockMovement(productID int, quantity int, movementType string, note string) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}

	// 1. Record movement
	_, err = tx.Exec(`
		INSERT INTO stock_movements (product_id, quantity, movement_type, note)
		VALUES (?, ?, ?, ?)
	`, productID, quantity, movementType, note)
	if err != nil {
		tx.Rollback()
		return err
	}

	// 2. Update actual product stock
	// If it's IN, we add. If OUT, we subtract?
	// Let's standardise: quantity in movement is raw.
	// If movementType is 'IN' or 'ADJUST' (positive), we add.
	// But simplify: The caller decides the sign of 'quantity'.
	// NO, typical stock logic:
	// IN: +qty
	// OUT: -qty
	// INVOICE: -qty
	// Let's assume the caller passes the signed quantity (e.g. -5 for invoice).
	// Or we handle it by type?
	// Let's stick to: The caller provides the signed integer change.

	_, err = tx.Exec(`UPDATE products SET stock = stock + ? WHERE id = ?`, quantity, productID)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

// RecordStockMovementTx allows recording within an existing transaction
func (s *Store) RecordStockMovementTx(tx *Transaction, productID int, quantity int, movementType string, note string) error {
	_, err := tx.Tx.Exec(`
		INSERT INTO stock_movements (product_id, quantity, movement_type, note)
		VALUES (?, ?, ?, ?)
	`, productID, quantity, movementType, note)
	if err != nil {
		return err
	}

	_, err = tx.Tx.Exec(`UPDATE products SET stock = stock + ? WHERE id = ?`, quantity, productID)
	return err
}

func (s *Store) ListStockMovements(productID int) ([]StockMovement, error) {
	rows, err := s.DB.Query(`
		SELECT id, product_id, quantity, movement_type, note, created_at
		FROM stock_movements
		WHERE product_id = ?
		ORDER BY created_at DESC
	`, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var movements []StockMovement
	for rows.Next() {
		var m StockMovement
		if err := rows.Scan(&m.ID, &m.ProductID, &m.Quantity, &m.MovementType, &m.Note, &m.CreatedAt); err != nil {
			return nil, err
		}
		movements = append(movements, m)
	}
	return movements, nil
}
