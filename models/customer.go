package models

import "time"

type Customer struct {
	ID             int
	CustomerNumber string
	Name           string
	Address        string
	Email          string
	CreatedAt      time.Time
}

func (s *Store) CreateCustomer(c Customer) (int, error) {
	res, err := s.DB.Exec(`
		INSERT INTO customers (customer_number, name, address, email)
		VALUES (?, ?, ?, ?)
	`, c.CustomerNumber, c.Name, c.Address, c.Email)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return int(id), err
}

func (s *Store) UpdateCustomer(c Customer) error {
	_, err := s.DB.Exec(`
		UPDATE customers
		SET customer_number = ?, name = ?, address = ?, email = ?
		WHERE id = ?
	`, c.CustomerNumber, c.Name, c.Address, c.Email, c.ID)
	return err
}

func (s *Store) DeleteCustomer(id int) error {
	_, err := s.DB.Exec("DELETE FROM customers WHERE id = ?", id)
	return err
}

func (s *Store) ListCustomers() ([]Customer, error) {
	rows, err := s.DB.Query(`SELECT id, customer_number, name, address, email, created_at FROM customers ORDER BY name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var customers []Customer
	for rows.Next() {
		var c Customer
		if err := rows.Scan(&c.ID, &c.CustomerNumber, &c.Name, &c.Address, &c.Email, &c.CreatedAt); err != nil {
			return nil, err
		}
		customers = append(customers, c)
	}
	return customers, nil
}

func (s *Store) GetCustomer(id int) (*Customer, error) {
	var c Customer
	err := s.DB.QueryRow(`
		SELECT id, customer_number, name, address, email, created_at
		FROM customers WHERE id = ?
	`, id).Scan(&c.ID, &c.CustomerNumber, &c.Name, &c.Address, &c.Email, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}
