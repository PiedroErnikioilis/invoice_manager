package db

import (
	"database/sql"
	"log"

	_ "modernc.org/sqlite"
)

func Init(dataSourceName string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dataSourceName)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	createTables := `
	CREATE TABLE IF NOT EXISTS invoices (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		invoice_number TEXT NOT NULL,
		date TEXT NOT NULL,
		sender_name TEXT NOT NULL,
		sender_address TEXT NOT NULL,
		recipient_name TEXT NOT NULL,
		recipient_address TEXT NOT NULL,
		tax_rate REAL DEFAULT 19.0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		status TEXT DEFAULT 'Entwurf',
		is_small_business BOOLEAN DEFAULT 0,
		customer_id INTEGER
	);

	CREATE TABLE IF NOT EXISTS invoice_items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		invoice_id INTEGER NOT NULL,
		description TEXT NOT NULL,
		quantity INTEGER NOT NULL,
		price_per_unit REAL NOT NULL,
		product_id INTEGER,
		FOREIGN KEY (invoice_id) REFERENCES invoices(id)
	);

	CREATE TABLE IF NOT EXISTS products (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		description TEXT,
		price REAL DEFAULT 0.0,
		stock INTEGER DEFAULT 0,
		unit TEXT DEFAULT 'Stk'
	);
	
	CREATE TABLE IF NOT EXISTS stock_movements (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		product_id INTEGER NOT NULL,
		quantity INTEGER NOT NULL,
		movement_type TEXT NOT NULL,
		note TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (product_id) REFERENCES products(id)
	);

	CREATE TABLE IF NOT EXISTS customers (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		address TEXT NOT NULL,
		email TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT
	);

	CREATE TABLE IF NOT EXISTS expenses (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		description TEXT NOT NULL,
		amount REAL NOT NULL,
		date TEXT NOT NULL,
		category TEXT,
		receipt_path TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`

	_, err = db.Exec(createTables)
	if err != nil {
		log.Printf("Error creating tables: %q", err)
		return nil, err
	}

	// Migrations for existing DBs
	migrations := []string{
		"ALTER TABLE invoices ADD COLUMN status TEXT DEFAULT 'Entwurf'",
		"ALTER TABLE invoices ADD COLUMN is_small_business BOOLEAN DEFAULT 0",
		"ALTER TABLE invoice_items ADD COLUMN product_id INTEGER",
		"ALTER TABLE invoices ADD COLUMN customer_id INTEGER",
	}

	for _, m := range migrations {
		_, _ = db.Exec(m)
	}

	return db, nil
}
