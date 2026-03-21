package models

import (
	"log/slog"
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
	slog.Debug("Recording stock movement", "product_id", productID, "quantity", quantity, "type", movementType, "note", note)
	tx, err := s.DB.Begin()
	if err != nil {
		slog.Error("Failed to begin transaction for stock movement", "error", err)
		return err
	}

	// 1. Record movement
	_, err = tx.Exec(`
		INSERT INTO stock_movements (product_id, quantity, movement_type, note)
		VALUES (?, ?, ?, ?)
	`, productID, quantity, movementType, note)
	if err != nil {
		slog.Error("Failed to insert stock movement", "product_id", productID, "error", err)
		tx.Rollback()
		return err
	}

	// 2. Update actual product stock
	_, err = tx.Exec(`UPDATE products SET stock = stock + ? WHERE id = ?`, quantity, productID)
	if err != nil {
		slog.Error("Failed to update product stock", "product_id", productID, "error", err)
		tx.Rollback()
		return err
	}

	// 3. Get new stock level for logging
	var newStock int
	_ = tx.QueryRow(`SELECT stock FROM products WHERE id = ?`, productID).Scan(&newStock)

	if err := tx.Commit(); err != nil {
		slog.Error("Failed to commit stock movement", "error", err)
		return err
	}

	slog.Info("Stock movement recorded", "product_id", productID, "quantity", quantity, "type", movementType, "new_stock", newStock)
	return nil
}

// RecordStockMovementTx allows recording within an existing transaction
func (s *Store) RecordStockMovementTx(tx *Transaction, productID int, quantity int, movementType string, note string) error {
	slog.Debug("Recording stock movement in TX", "product_id", productID, "quantity", quantity, "type", movementType, "note", note)
	_, err := tx.Tx.Exec(`
		INSERT INTO stock_movements (product_id, quantity, movement_type, note)
		VALUES (?, ?, ?, ?)
	`, productID, quantity, movementType, note)
	if err != nil {
		slog.Error("Failed to insert stock movement in TX", "product_id", productID, "error", err)
		return err
	}

	_, err = tx.Tx.Exec(`UPDATE products SET stock = stock + ? WHERE id = ?`, quantity, productID)
	if err != nil {
		slog.Error("Failed to update product stock in TX", "product_id", productID, "error", err)
		return err
	}

	// Get new stock level for logging (in TX)
	var newStock int
	_ = tx.Tx.QueryRow(`SELECT stock FROM products WHERE id = ?`, productID).Scan(&newStock)

	slog.Debug("Stock movement recorded in TX", "product_id", productID, "quantity", quantity, "type", movementType, "new_stock_in_tx", newStock)
	return nil
}

func (s *Store) ListStockMovements(productID int) ([]StockMovement, error) {
	rows, err := s.DB.Query(`
		SELECT id, product_id, quantity, movement_type, note, created_at
		FROM stock_movements
		WHERE product_id = ?
		ORDER BY created_at DESC
	`, productID)
	if err != nil {
		slog.Error("Failed to list stock movements", "product_id", productID, "error", err)
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
