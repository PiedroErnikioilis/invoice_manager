package models

import (
	"database/sql"
	"fmt"
	"time"
)

type ExpenseCategory struct {
	ID   int
	Name string
}

type Expense struct {
	ID           int
	Description  string
	Amount       float64 // Brutto
	TaxRate      float64
	Date         string
	CategoryID   *int
	CategoryName string // joined from expense_categories
	ReceiptPath  string
	ReceiptData  string
	CreatedAt    time.Time
}

func (e *Expense) Net() float64 {
	return e.Amount / (1 + e.TaxRate/100)
}

func (e *Expense) Tax() float64 {
	return e.Amount - e.Net()
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
		INSERT INTO expenses (description, amount, date, tax_rate, category_id, receipt_path, receipt_data)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, e.Description, e.Amount, e.Date, e.TaxRate, e.CategoryID, e.ReceiptPath, e.ReceiptData)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return int(id), err
}

func (s *Store) ListExpenses(year ...int) ([]Expense, error) {
	query := `
		SELECT e.id, e.description, e.amount, e.date, e.tax_rate, e.category_id, COALESCE(ec.name, ''), e.receipt_path, e.created_at
		FROM expenses e
		LEFT JOIN expense_categories ec ON e.category_id = ec.id`
	if len(year) > 0 && year[0] > 0 {
		query += fmt.Sprintf(" WHERE e.date LIKE '%d-%%'", year[0])
	}
	query += ` ORDER BY e.date DESC`
	rows, err := s.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var expenses []Expense
	for rows.Next() {
		var e Expense
		if err := rows.Scan(&e.ID, &e.Description, &e.Amount, &e.Date, &e.TaxRate, &e.CategoryID, &e.CategoryName, &e.ReceiptPath, &e.CreatedAt); err != nil {
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

func (s *Store) GetExpense(id int) (Expense, error) {
	var e Expense
	err := s.DB.QueryRow(`
		SELECT e.id, e.description, e.amount, e.date, e.tax_rate, e.category_id, COALESCE(ec.name, ''), e.receipt_path, e.receipt_data, e.created_at
		FROM expenses e
		LEFT JOIN expense_categories ec ON e.category_id = ec.id
		WHERE e.id = ?
	`, id).Scan(&e.ID, &e.Description, &e.Amount, &e.Date, &e.TaxRate, &e.CategoryID, &e.CategoryName, &e.ReceiptPath, &e.ReceiptData, &e.CreatedAt)
	return e, err
}

func (s *Store) UpdateExpense(e Expense) error {
	query := `
		UPDATE expenses 
		SET description = ?, amount = ?, date = ?, tax_rate = ?, category_id = ?
	`
	args := []interface{}{e.Description, e.Amount, e.Date, e.TaxRate, e.CategoryID}

	if e.ReceiptData != "" {
		query += ", receipt_path = ?, receipt_data = ?"
		args = append(args, e.ReceiptPath, e.ReceiptData)
	}

	query += " WHERE id = ?"
	args = append(args, e.ID)

	_, err := s.DB.Exec(query, args...)
	return err
}

func (s *Store) DeleteExpense(id int) error {
	_, err := s.DB.Exec("DELETE FROM expenses WHERE id = ?", id)
	return err
}
