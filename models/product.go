package models

import "log/slog"

type Product struct {
	ID          int
	Name        string
	Description string
	Price       float64
	Stock       int
	MinStock    int
	Unit        string
}

func (s *Store) CreateProduct(p Product) (int, error) {
	slog.Info("Creating product", "name", p.Name, "price", p.Price)
	res, err := s.DB.Exec(`
		INSERT INTO products (name, description, price, stock, min_stock, unit)
		VALUES (?, ?, ?, ?, ?, ?)
	`, p.Name, p.Description, p.Price, p.Stock, p.MinStock, p.Unit)
	if err != nil {
		slog.Error("Failed to insert product", "name", p.Name, "error", err)
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		slog.Error("Failed to get last insert id for product", "error", err)
		return 0, err
	}
	slog.Info("Product created successfully", "id", id)
	return int(id), nil
}

func (s *Store) UpdateProduct(p Product) error {
	slog.Info("Updating product", "id", p.ID, "name", p.Name)
	_, err := s.DB.Exec(`
		UPDATE products
		SET name = ?, description = ?, price = ?, stock = ?, min_stock = ?, unit = ?
		WHERE id = ?
	`, p.Name, p.Description, p.Price, p.Stock, p.MinStock, p.Unit, p.ID)
	if err != nil {
		slog.Error("Failed to update product", "id", p.ID, "error", err)
		return err
	}
	slog.Info("Product updated successfully", "id", p.ID)
	return nil
}

func (s *Store) DeleteProduct(id int) error {
	slog.Info("Deleting product", "id", id)
	_, err := s.DB.Exec(`DELETE FROM products WHERE id = ?`, id)
	if err != nil {
		slog.Error("Failed to delete product", "id", id, "error", err)
		return err
	}
	slog.Info("Product deleted successfully", "id", id)
	return nil
}

func (s *Store) ListProducts() ([]Product, error) {
	slog.Debug("Listing products from database")
	rows, err := s.DB.Query(`SELECT id, name, description, price, stock, min_stock, unit FROM products ORDER BY name ASC`)
	if err != nil {
		slog.Error("Failed to query products", "error", err)
		return nil, err
	}
	defer rows.Close()

	var products []Product
	for rows.Next() {
		var p Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Price, &p.Stock, &p.MinStock, &p.Unit); err != nil {
			slog.Error("Failed to scan product row", "error", err)
			return nil, err
		}
		products = append(products, p)
	}
	return products, nil
}

func (s *Store) GetProduct(id int) (*Product, error) {
	slog.Debug("Getting product details", "id", id)
	var p Product
	err := s.DB.QueryRow(`
		SELECT id, name, description, price, stock, min_stock, unit
		FROM products WHERE id = ?
	`, id).Scan(&p.ID, &p.Name, &p.Description, &p.Price, &p.Stock, &p.MinStock, &p.Unit)
	if err != nil {
		slog.Error("Failed to get product", "id", id, "error", err)
		return nil, err
	}
	return &p, nil
}
