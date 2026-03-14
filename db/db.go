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

	// Enable foreign key enforcement
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, err
	}

	createTables := `
	CREATE TABLE IF NOT EXISTS customers (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		address TEXT NOT NULL,
		email TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

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
		status TEXT DEFAULT 'Entwurf' CHECK(status IN ('Entwurf', 'Offen', 'Bezahlt', 'Storniert')),
		is_small_business BOOLEAN DEFAULT 0,
		customer_id INTEGER REFERENCES customers(id) ON DELETE SET NULL
	);

	CREATE TABLE IF NOT EXISTS products (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		description TEXT,
		price REAL DEFAULT 0.0,
		stock INTEGER DEFAULT 0,
		min_stock INTEGER DEFAULT 0,
		unit TEXT DEFAULT 'Stk'
	);

	CREATE TABLE IF NOT EXISTS invoice_items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		invoice_id INTEGER NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
		description TEXT NOT NULL,
		quantity INTEGER NOT NULL,
		price_per_unit REAL NOT NULL,
		product_id INTEGER REFERENCES products(id) ON DELETE SET NULL
	);

	CREATE TABLE IF NOT EXISTS stock_movements (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		product_id INTEGER NOT NULL REFERENCES products(id) ON DELETE CASCADE,
		quantity INTEGER NOT NULL,
		movement_type TEXT NOT NULL CHECK(movement_type IN ('INVOICE', 'INVOICE_UPDATE', 'PURCHASE', 'MANUAL_ADD', 'MANUAL_REMOVE', 'CANCELLATION')),
		note TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT
	);

	CREATE TABLE IF NOT EXISTS expense_categories (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE
	);

	CREATE TABLE IF NOT EXISTS expenses (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		description TEXT NOT NULL,
		amount REAL NOT NULL,
		date TEXT NOT NULL,
		tax_rate REAL DEFAULT 19.0,
		category_id INTEGER REFERENCES expense_categories(id) ON DELETE SET NULL,
		receipt_path TEXT,
		receipt_data TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS recurring_expenses (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		description TEXT NOT NULL,
		amount REAL NOT NULL,
		tax_rate REAL DEFAULT 19.0,
		interval TEXT NOT NULL CHECK(interval IN ('monthly', 'quarterly', 'yearly')),
		category_id INTEGER REFERENCES expense_categories(id) ON DELETE SET NULL,
		start_date TEXT NOT NULL,
		last_booked_at TEXT,
		is_active BOOLEAN DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`

	_, err = db.Exec(createTables)
	if err != nil {
		log.Printf("Error creating tables: %q", err)
		return nil, err
	}

	// Migrations for existing DBs (errors are ignored for already-applied migrations)
	migrations := []string{
		"ALTER TABLE invoices ADD COLUMN status TEXT DEFAULT 'Entwurf'",
		"ALTER TABLE invoices ADD COLUMN is_small_business BOOLEAN DEFAULT 0",
		"ALTER TABLE invoice_items ADD COLUMN product_id INTEGER",
		"ALTER TABLE invoices ADD COLUMN customer_id INTEGER",
		"ALTER TABLE expenses ADD COLUMN receipt_data TEXT",
		"ALTER TABLE expenses ADD COLUMN category_id INTEGER REFERENCES expense_categories(id) ON DELETE SET NULL",
		"ALTER TABLE expenses ADD COLUMN tax_rate REAL DEFAULT 19.0",
		"ALTER TABLE products ADD COLUMN min_stock INTEGER DEFAULT 0",
	}

	for _, m := range migrations {
		_, _ = db.Exec(m)
	}

	// Migrate existing category text values into expense_categories table
	migrateCategories(db)

	return db, nil
}

// migrateCategories moves existing category text values from expenses
// into the expense_categories table and sets the category_id foreign key.
func migrateCategories(db *sql.DB) {
	// Check if the old category column exists
	rows, err := db.Query(`SELECT DISTINCT category FROM expenses WHERE category IS NOT NULL AND category != '' AND (category_id IS NULL OR category_id = 0)`)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			continue
		}
		// Insert category if not exists
		db.Exec(`INSERT OR IGNORE INTO expense_categories (name) VALUES (?)`, name)

		// Get the category ID
		var catID int
		if err := db.QueryRow(`SELECT id FROM expense_categories WHERE name = ?`, name).Scan(&catID); err != nil {
			continue
		}

		// Update expenses with this category
		db.Exec(`UPDATE expenses SET category_id = ? WHERE category = ? AND (category_id IS NULL OR category_id = 0)`, catID, name)
	}
}
