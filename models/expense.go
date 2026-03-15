package models

import (
	"database/sql"
	"fmt"
	"log/slog"
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

	slog.Info("Creating new expense category", "name", name)
	res, err := s.DB.Exec(`INSERT INTO expense_categories (name) VALUES (?)`, name)
	if err != nil {
		slog.Error("Failed to create expense category", "name", name, "error", err)
		return 0, err
	}
	insertedID, err := res.LastInsertId()
	if err != nil {
		slog.Error("Failed to get last insert id for expense category", "error", err)
		return 0, err
	}
	return int(insertedID), nil
}

func (s *Store) ListExpenseCategories() ([]ExpenseCategory, error) {
	slog.Debug("Listing expense categories from database")
	rows, err := s.DB.Query(`SELECT id, name FROM expense_categories ORDER BY name`)
	if err != nil {
		slog.Error("Failed to query expense categories", "error", err)
		return nil, err
	}
	defer rows.Close()

	var categories []ExpenseCategory
	for rows.Next() {
		var c ExpenseCategory
		if err := rows.Scan(&c.ID, &c.Name); err != nil {
			slog.Error("Failed to scan expense category row", "error", err)
			return nil, err
		}
		categories = append(categories, c)
	}
	return categories, nil
}

func (s *Store) CreateExpense(e Expense) (int, error) {
	slog.Info("Creating expense", "description", e.Description, "amount", e.Amount, "date", e.Date)
	res, err := s.DB.Exec(`
		INSERT INTO expenses (description, amount, date, tax_rate, category_id, receipt_path, receipt_data)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, e.Description, e.Amount, e.Date, e.TaxRate, e.CategoryID, e.ReceiptPath, e.ReceiptData)
	if err != nil {
		slog.Error("Failed to insert expense", "description", e.Description, "error", err)
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		slog.Error("Failed to get last insert id for expense", "error", err)
		return 0, err
	}
	slog.Info("Expense created successfully", "id", id)
	return int(id), nil
}

func (s *Store) ListExpenses(year ...int) ([]Expense, error) {
	y := 0
	if len(year) > 0 {
		y = year[0]
	}
	slog.Debug("Listing expenses from database", "year", y)
	query := `
		SELECT e.id, e.description, e.amount, e.date, e.tax_rate, e.category_id, COALESCE(ec.name, ''), e.receipt_path, e.created_at
		FROM expenses e
		LEFT JOIN expense_categories ec ON e.category_id = ec.id`
	if y > 0 {
		query += fmt.Sprintf(" WHERE e.date LIKE '%d-%%'", y)
	}
	query += ` ORDER BY e.date DESC`
	rows, err := s.DB.Query(query)
	if err != nil {
		slog.Error("Failed to query expenses", "year", y, "error", err)
		return nil, err
	}
	defer rows.Close()

	var expenses []Expense
	for rows.Next() {
		var e Expense
		if err := rows.Scan(&e.ID, &e.Description, &e.Amount, &e.Date, &e.TaxRate, &e.CategoryID, &e.CategoryName, &e.ReceiptPath, &e.CreatedAt); err != nil {
			slog.Error("Failed to scan expense row", "error", err)
			return nil, err
		}
		expenses = append(expenses, e)
	}
	return expenses, nil
}

func (s *Store) GetExpenseReceipt(id int) (string, string, error) {
	slog.Debug("Getting expense receipt details", "id", id)
	var path string
	var data sql.NullString
	err := s.DB.QueryRow(`SELECT receipt_path, receipt_data FROM expenses WHERE id = ?`, id).Scan(&path, &data)
	if err != nil {
		slog.Error("Failed to get expense receipt", "id", id, "error", err)
		return "", "", err
	}
	return path, data.String, nil
}

func (s *Store) GetExpense(id int) (Expense, error) {
	slog.Debug("Getting expense details", "id", id)
	var e Expense
	err := s.DB.QueryRow(`
		SELECT e.id, e.description, e.amount, e.date, e.tax_rate, e.category_id, COALESCE(ec.name, ''), e.receipt_path, e.receipt_data, e.created_at
		FROM expenses e
		LEFT JOIN expense_categories ec ON e.category_id = ec.id
		WHERE e.id = ?
	`, id).Scan(&e.ID, &e.Description, &e.Amount, &e.Date, &e.TaxRate, &e.CategoryID, &e.CategoryName, &e.ReceiptPath, &e.ReceiptData, &e.CreatedAt)
	if err != nil {
		slog.Error("Failed to get expense", "id", id, "error", err)
	}
	return e, err
}

func (s *Store) UpdateExpense(e Expense) error {
	slog.Info("Updating expense", "id", e.ID, "description", e.Description)
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
	if err != nil {
		slog.Error("Failed to update expense", "id", e.ID, "error", err)
		return err
	}
	slog.Info("Expense updated successfully", "id", e.ID)
	return nil
}

type RecurringExpense struct {
	ID           int
	Description  string
	Amount       float64
	TaxRate      float64
	Interval     string // monthly, quarterly, yearly
	CategoryID   *int
	CategoryName string
	StartDate    string
	LastBookedAt string
	IsActive     bool
	CreatedAt    time.Time
}

func (s *Store) CreateRecurringExpense(re RecurringExpense) (int, error) {
	slog.Info("Creating recurring expense", "description", re.Description, "amount", re.Amount, "interval", re.Interval)
	res, err := s.DB.Exec(`
		INSERT INTO recurring_expenses (description, amount, tax_rate, interval, category_id, start_date, last_booked_at, is_active)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, re.Description, re.Amount, re.TaxRate, re.Interval, re.CategoryID, re.StartDate, re.LastBookedAt, re.IsActive)
	if err != nil {
		slog.Error("Failed to insert recurring expense", "description", re.Description, "error", err)
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		slog.Error("Failed to get last insert id for recurring expense", "error", err)
		return 0, err
	}
	slog.Info("Recurring expense created successfully", "id", id)
	return int(id), nil
}

func (s *Store) ListRecurringExpenses() ([]RecurringExpense, error) {
	slog.Debug("Listing recurring expenses from database")
	rows, err := s.DB.Query(`
		SELECT re.id, re.description, re.amount, re.tax_rate, re.interval, re.category_id, COALESCE(ec.name, ''), re.start_date, COALESCE(re.last_booked_at, ''), re.is_active, re.created_at
		FROM recurring_expenses re
		LEFT JOIN expense_categories ec ON re.category_id = ec.id
		ORDER BY re.description ASC
	`)
	if err != nil {
		slog.Error("Failed to query recurring expenses", "error", err)
		return nil, err
	}
	defer rows.Close()

	var results []RecurringExpense
	for rows.Next() {
		var re RecurringExpense
		if err := rows.Scan(&re.ID, &re.Description, &re.Amount, &re.TaxRate, &re.Interval, &re.CategoryID, &re.CategoryName, &re.StartDate, &re.LastBookedAt, &re.IsActive, &re.CreatedAt); err != nil {
			slog.Error("Failed to scan recurring expense row", "error", err)
			return nil, err
		}
		results = append(results, re)
	}
	return results, nil
}

func (s *Store) DeleteRecurringExpense(id int) error {
	slog.Info("Deleting recurring expense", "id", id)
	_, err := s.DB.Exec(`DELETE FROM recurring_expenses WHERE id = ?`, id)
	if err != nil {
		slog.Error("Failed to delete recurring expense", "id", id, "error", err)
		return err
	}
	slog.Info("Recurring expense deleted successfully", "id", id)
	return nil
}

func (s *Store) UpdateRecurringExpense(re RecurringExpense) error {
	slog.Debug("Updating recurring expense record", "id", re.ID)
	_, err := s.DB.Exec(`
		UPDATE recurring_expenses
		SET description = ?, amount = ?, tax_rate = ?, interval = ?, category_id = ?, start_date = ?, last_booked_at = ?, is_active = ?
		WHERE id = ?
	`, re.Description, re.Amount, re.TaxRate, re.Interval, re.CategoryID, re.StartDate, re.LastBookedAt, re.IsActive, re.ID)
	if err != nil {
		slog.Error("Failed to update recurring expense record", "id", re.ID, "error", err)
		return err
	}
	return nil
}

func (s *Store) ProcessRecurringExpenses() error {
	slog.Debug("Processing recurring expenses")
	recurring, err := s.ListRecurringExpenses()
	if err != nil {
		return err
	}

	today := time.Now()

	for _, re := range recurring {
		if !re.IsActive {
			continue
		}

		startDate, _ := time.Parse("2006-01-02", re.StartDate)
		lastBooked := startDate.AddDate(0, 0, -1) // Default to day before start
		if re.LastBookedAt != "" {
			lastBooked, _ = time.Parse("2006-01-02", re.LastBookedAt)
		}

		// Calculate next due date
		nextDue := lastBooked
		for {
			switch re.Interval {
			case "monthly":
				nextDue = nextDue.AddDate(0, 1, 0)
			case "quarterly":
				nextDue = nextDue.AddDate(0, 3, 0)
			case "yearly":
				nextDue = nextDue.AddDate(1, 0, 0)
			default:
				slog.Error("Invalid interval for recurring expense", "id", re.ID, "interval", re.Interval)
				goto next_re // Invalid interval
			}

			// If nextDue is in the future, we are done for this RE
			if nextDue.After(today) {
				break
			}

			// Book it!
			slog.Info("Booking recurring expense", "id", re.ID, "description", re.Description, "date", nextDue.Format("2006-01-02"))
			expense := Expense{
				Description: re.Description + " (automatisch)",
				Amount:      re.Amount,
				TaxRate:     re.TaxRate,
				Date:        nextDue.Format("2006-01-02"),
				CategoryID:  re.CategoryID,
			}

			_, err := s.CreateExpense(expense)
			if err != nil {
				slog.Error("Failed to create expense from recurring", "id", re.ID, "error", err)
				return err
			}

			// Update LastBookedAt in DB
			re.LastBookedAt = nextDue.Format("2006-01-02")
			err = s.UpdateRecurringExpense(re)
			if err != nil {
				slog.Error("Failed to update last_booked_at for recurring expense", "id", re.ID, "error", err)
				return err
			}
		}
	next_re:
	}

	return nil
}

func (s *Store) DeleteExpense(id int) error {
	slog.Info("Deleting expense", "id", id)
	_, err := s.DB.Exec("DELETE FROM expenses WHERE id = ?", id)
	if err != nil {
		slog.Error("Failed to delete expense", "id", id, "error", err)
		return err
	}
	slog.Info("Expense deleted successfully", "id", id)
	return nil
}
