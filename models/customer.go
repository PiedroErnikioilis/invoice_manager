package models

import (
	"log/slog"
	"time"
)

type Customer struct {
	ID             int
	CustomerNumber string
	Name           string
	Address        string
	Email          string
	CreatedAt      time.Time
}

func (s *Store) CreateCustomer(c Customer) (int, error) {
	slog.Info("Creating customer", "name", c.Name, "customer_number", c.CustomerNumber)
	res, err := s.DB.Exec(`
		INSERT INTO customers (customer_number, name, address, email)
		VALUES (?, ?, ?, ?)
	`, c.CustomerNumber, c.Name, c.Address, c.Email)
	if err != nil {
		slog.Error("Failed to insert customer", "name", c.Name, "error", err)
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		slog.Error("Failed to get last insert id for customer", "error", err)
		return 0, err
	}
	slog.Info("Customer created successfully", "id", id, "customer_number", c.CustomerNumber)
	return int(id), err
}

func (s *Store) UpdateCustomer(c Customer) error {
	slog.Info("Updating customer", "id", c.ID, "name", c.Name)
	_, err := s.DB.Exec(`
		UPDATE customers
		SET customer_number = ?, name = ?, address = ?, email = ?
		WHERE id = ?
	`, c.CustomerNumber, c.Name, c.Address, c.Email, c.ID)
	if err != nil {
		slog.Error("Failed to update customer", "id", c.ID, "error", err)
		return err
	}
	slog.Info("Customer updated successfully", "id", c.ID)
	return nil
}

func (s *Store) DeleteCustomer(id int) error {
	slog.Info("Deleting customer", "id", id)
	_, err := s.DB.Exec("DELETE FROM customers WHERE id = ?", id)
	if err != nil {
		slog.Error("Failed to delete customer", "id", id, "error", err)
		return err
	}
	slog.Info("Customer deleted successfully", "id", id)
	return nil
}

func (s *Store) ListCustomers() ([]Customer, error) {
	slog.Debug("Listing customers from database")
	rows, err := s.DB.Query(`SELECT id, customer_number, name, address, email, created_at FROM customers ORDER BY name ASC`)
	if err != nil {
		slog.Error("Failed to query customers", "error", err)
		return nil, err
	}
	defer rows.Close()

	var customers []Customer
	for rows.Next() {
		var c Customer
		if err := rows.Scan(&c.ID, &c.CustomerNumber, &c.Name, &c.Address, &c.Email, &c.CreatedAt); err != nil {
			slog.Error("Failed to scan customer row", "error", err)
			return nil, err
		}
		customers = append(customers, c)
	}
	return customers, nil
}

func (s *Store) GetCustomer(id int) (*Customer, error) {
	slog.Debug("Getting customer details", "id", id)
	var c Customer
	err := s.DB.QueryRow(`
		SELECT id, customer_number, name, address, email, created_at
		FROM customers WHERE id = ?
	`, id).Scan(&c.ID, &c.CustomerNumber, &c.Name, &c.Address, &c.Email, &c.CreatedAt)
	if err != nil {
		slog.Error("Failed to get customer", "id", id, "error", err)
		return nil, err
	}
	return &c, nil
}
