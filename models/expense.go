package models

import (
	"database/sql"
	"time"
)

type ExpenseCategory struct {
	ID   int
	Name string
}

type Expense struct {
	ID           int
	Description  string
	Amount       float64
	Date         string
	CategoryID   *int
	CategoryName string // joined from expense_categories
	ReceiptPath  string
	ReceiptData  string
	CreatedAt    time.Time
}

func (s *Store) CreateExpenseCategory(name string) (int, error) {
	// Return existing category if it already exists
	var id int
	err := s.DB.QueryRow(`SELECT id FROM expense_categories WHERE name = ?`, name).Scan(&id)
	if err == nil {
		return id, nil
	}

	res, err := s.DB.Exec(`INSERT INTO expense_categories (name) VALUES (?)`, name)
	if err != nil {
		return 0, err
	}
	insertedID, err := res.LastInsertId()
	return int(insertedID), err
}

func (s *Store) ListExpenseCategories() ([]ExpenseCategory, error) {
	rows, err := s.DB.Query(`SELECT id, name FROM expense_categories ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []ExpenseCategory
	for rows.Next() {
		var c ExpenseCategory
		if err := rows.Scan(&c.ID, &c.Name); err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}
	return categories, nil
}

func (s *Store) CreateExpense(e Expense) (int, error) {
	res, err := s.DB.Exec(`
		INSERT INTO expenses (description, amount, date, category_id, receipt_path, receipt_data)
		VALUES (?, ?, ?, ?, ?, ?)
	`, e.Description, e.Amount, e.Date, e.CategoryID, e.ReceiptPath, e.ReceiptData)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return int(id), err
}

func (s *Store) ListExpenses() ([]Expense, error) {
	rows, err := s.DB.Query(`
		SELECT e.id, e.description, e.amount, e.date, e.category_id, COALESCE(ec.name, ''), e.receipt_path, e.created_at
		FROM expenses e
		LEFT JOIN expense_categories ec ON e.category_id = ec.id
		ORDER BY e.date DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var expenses []Expense
	for rows.Next() {
		var e Expense
		if err := rows.Scan(&e.ID, &e.Description, &e.Amount, &e.Date, &e.CategoryID, &e.CategoryName, &e.ReceiptPath, &e.CreatedAt); err != nil {
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
