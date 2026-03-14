package models

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
	res, err := s.DB.Exec(`
		INSERT INTO products (name, description, price, stock, min_stock, unit)
		VALUES (?, ?, ?, ?, ?, ?)
	`, p.Name, p.Description, p.Price, p.Stock, p.MinStock, p.Unit)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return int(id), err
}

func (s *Store) UpdateProduct(p Product) error {
	_, err := s.DB.Exec(`
		UPDATE products
		SET name = ?, description = ?, price = ?, stock = ?, min_stock = ?, unit = ?
		WHERE id = ?
	`, p.Name, p.Description, p.Price, p.Stock, p.MinStock, p.Unit, p.ID)
	return err
}

func (s *Store) DeleteProduct(id int) error {
	_, err := s.DB.Exec(`DELETE FROM products WHERE id = ?`, id)
	return err
}

func (s *Store) ListProducts() ([]Product, error) {
	rows, err := s.DB.Query(`SELECT id, name, description, price, stock, min_stock, unit FROM products ORDER BY name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []Product
	for rows.Next() {
		var p Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Price, &p.Stock, &p.MinStock, &p.Unit); err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	return products, nil
}

func (s *Store) GetProduct(id int) (*Product, error) {
	var p Product
	err := s.DB.QueryRow(`
		SELECT id, name, description, price, stock, min_stock, unit
		FROM products WHERE id = ?
	`, id).Scan(&p.ID, &p.Name, &p.Description, &p.Price, &p.Stock, &p.MinStock, &p.Unit)
	if err != nil {
		return nil, err
	}
	return &p, nil
}
