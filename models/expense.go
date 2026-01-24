package models

import (
	"database/sql"
	"time"
)

type Expense struct {
	ID          int
	Description string
	Amount      float64
	Date        string
	Category    string
	ReceiptPath string
	ReceiptData string
	CreatedAt   time.Time
}

func (s *Store) CreateExpense(e Expense) (int, error) {
	res, err := s.DB.Exec(`
		INSERT INTO expenses (description, amount, date, category, receipt_path, receipt_data)
		VALUES (?, ?, ?, ?, ?, ?)
	`, e.Description, e.Amount, e.Date, e.Category, e.ReceiptPath, e.ReceiptData)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return int(id), err
}

func (s *Store) ListExpenses() ([]Expense, error) {
	rows, err := s.DB.Query(`SELECT id, description, amount, date, category, receipt_path, created_at FROM expenses ORDER BY date DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var expenses []Expense
	for rows.Next() {
		var e Expense
		if err := rows.Scan(&e.ID, &e.Description, &e.Amount, &e.Date, &e.Category, &e.ReceiptPath, &e.CreatedAt); err != nil {
			return nil, err
		}
		expenses = append(expenses, e)
	}
	return expenses, nil
}

func (s *Store) GetExpenseReceipt(id int) (string, string, error) {
	var path string
	var data sql.NullString
	err := s.DB.QueryRow(`SELECT receipt_path, receipt_data FROM expenses WHERE id = ?`, id).Scan(&path, &data)
	if err != nil {
		return "", "", err
	}
	return path, data.String, nil
}

func (s *Store) DeleteExpense(id int) error {
	_, err := s.DB.Exec("DELETE FROM expenses WHERE id = ?", id)
	return err
}
