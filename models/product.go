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
	slog.Debug("Executing CreateProduct", "name", p.Name, "price", p.Price)
	query := `
		INSERT INTO products (name, description, price, stock, min_stock, unit)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	slog.Debug("Inserting product into database", "name", p.Name, "query", query)
	res, err := s.DB.Exec(query, p.Name, p.Description, p.Price, p.Stock, p.MinStock, p.Unit)
	if err != nil {
		slog.Error("Failed to insert product", "name", p.Name, "error", err)
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		slog.Error("Failed to get last insert id for product", "error", err)
		return 0, err
	}
	slog.Info("CreateProduct completed successfully", "id", id, "name", p.Name)
	return int(id), nil
}

func (s *Store) UpdateProduct(p Product) error {
	slog.Debug("Executing UpdateProduct", "id", p.ID, "name", p.Name)
	query := `
		UPDATE products
		SET name = ?, description = ?, price = ?, stock = ?, min_stock = ?, unit = ?
		WHERE id = ?
	`
	slog.Debug("Updating product in database", "id", p.ID, "query", query)
	_, err := s.DB.Exec(query, p.Name, p.Description, p.Price, p.Stock, p.MinStock, p.Unit, p.ID)
	if err != nil {
		slog.Error("Failed to update product", "id", p.ID, "error", err)
		return err
	}
	slog.Info("UpdateProduct completed successfully", "id", p.ID)
	return nil
}

func (s *Store) DeleteProduct(id int) error {
	slog.Debug("Executing DeleteProduct", "id", id)
	query := `DELETE FROM products WHERE id = ?`
	slog.Debug("Deleting product from database", "id", id, "query", query)
	_, err := s.DB.Exec(query, id)
	if err != nil {
		slog.Error("Failed to delete product", "id", id, "error", err)
		return err
	}
	slog.Info("DeleteProduct completed successfully", "id", id)
	return nil
}

func (s *Store) ListProducts() ([]Product, error) {
	slog.Debug("Executing ListProducts")
	query := `SELECT id, name, description, price, stock, min_stock, unit FROM products ORDER BY name ASC`
	slog.Debug("Querying products from database", "query", query)
	rows, err := s.DB.Query(query)
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
	slog.Info("ListProducts completed successfully", "count", len(products))
	return products, nil
}

func (s *Store) GetProduct(id int) (*Product, error) {
	slog.Debug("Executing GetProduct", "id", id)
	var p Product
	query := `
		SELECT id, name, description, price, stock, min_stock, unit
		FROM products WHERE id = ?
	`
	slog.Debug("Querying product details", "id", id, "query", query)
	err := s.DB.QueryRow(query, id).Scan(&p.ID, &p.Name, &p.Description, &p.Price, &p.Stock, &p.MinStock, &p.Unit)
	if err != nil {
		slog.Error("Failed to get product", "id", id, "error", err)
		return nil, err
	}
	slog.Info("GetProduct completed successfully", "id", id, "name", p.Name)
	return &p, nil
}
