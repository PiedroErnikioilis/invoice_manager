package db

import (
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const (
	defaultBackupPath             = "./backups"
	defaultBackupMinIntervalHours = 24
)

type backupSettings struct {
	backupDir   string
	autoBackup  bool
	minInterval int // Stunden
}

// readBackupSettings liest Backup-Einstellungen direkt aus der DB-Datei.
func readBackupSettings(dbPath string) backupSettings {
	s := backupSettings{
		backupDir:   defaultBackupPath,
		autoBackup:  true,
		minInterval: defaultBackupMinIntervalHours,
	}
	tmpDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return s
	}
	tmpDB.SetMaxOpenConns(1)
	defer tmpDB.Close()

	var val string
	if tmpDB.QueryRow("SELECT value FROM settings WHERE key = 'backup_path'").Scan(&val) == nil && val != "" {
		s.backupDir = val
	}
	if tmpDB.QueryRow("SELECT value FROM settings WHERE key = 'auto_backup_enabled'").Scan(&val) == nil {
		s.autoBackup = val != "false"
	}
	if tmpDB.QueryRow("SELECT value FROM settings WHERE key = 'backup_min_interval_hours'").Scan(&val) == nil && val != "" {
		fmt.Sscanf(val, "%d", &s.minInterval)
	}
	return s
}

// lastBackupAge gibt das Alter des neuesten Backups im Verzeichnis zurück.
func lastBackupAge(backupDir string) time.Duration {
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return time.Duration(1<<63 - 1) // Max duration → kein Backup vorhanden
	}
	var newest time.Time
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".db") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().After(newest) {
			newest = info.ModTime()
		}
	}
	if newest.IsZero() {
		return time.Duration(1<<63 - 1)
	}
	return time.Since(newest)
}

// PreMigrationBackup erstellt ein Backup der bestehenden DB bevor Migrationen laufen.
// Wird nur ausgeführt wenn die DB-Datei bereits existiert (Update-Fall).
func PreMigrationBackup(dbPath string) error {
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return nil // Neue Installation, kein Backup nötig
	}

	cfg := readBackupSettings(dbPath)
	if !cfg.autoBackup {
		return nil
	}

	if err := os.MkdirAll(cfg.backupDir, 0755); err != nil {
		return fmt.Errorf("backup-Verzeichnis erstellen: %w", err)
	}

	// Jahresabschluss-Backup prüfen
	createYearEndBackup(dbPath, cfg.backupDir)

	// Mindestzeitraum prüfen
	if cfg.minInterval > 0 {
		age := lastBackupAge(cfg.backupDir)
		if age < time.Duration(cfg.minInterval)*time.Hour {
			slog.Info("Pre-Migration-Backup übersprungen (letztes Backup zu aktuell)",
				"age", age.Round(time.Minute),
				"min_interval_hours", cfg.minInterval)
			return nil
		}
	}

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	baseName := fmt.Sprintf("pre_migration_%s", timestamp)

	if err := copyFileSimple(dbPath, filepath.Join(cfg.backupDir, baseName+".db")); err != nil {
		return fmt.Errorf("DB kopieren: %w", err)
	}
	copyFileSimple(dbPath+"-wal", filepath.Join(cfg.backupDir, baseName+".db-wal"))
	copyFileSimple(dbPath+"-shm", filepath.Join(cfg.backupDir, baseName+".db-shm"))

	slog.Info("Pre-Migration-Backup erstellt", "path", filepath.Join(cfg.backupDir, baseName+".db"))
	return nil
}

// createYearEndBackup prüft ob ein Jahresabschluss-Backup für das Vorjahr existiert.
// Falls nicht, wird eines erstellt.
func createYearEndBackup(dbPath, backupDir string) {
	now := time.Now()
	prevYear := now.Year() - 1
	expectedName := fmt.Sprintf("jahresabschluss_%d.db", prevYear)
	expectedPath := filepath.Join(backupDir, expectedName)

	if _, err := os.Stat(expectedPath); err == nil {
		return // Bereits vorhanden
	}

	if err := copyFileSimple(dbPath, expectedPath); err != nil {
		slog.Error("Jahresabschluss-Backup fehlgeschlagen", "error", err)
		return
	}
	copyFileSimple(dbPath+"-wal", expectedPath+"-wal")
	copyFileSimple(dbPath+"-shm", expectedPath+"-shm")

	slog.Info("Jahresabschluss-Backup erstellt", "path", expectedPath)
}

func copyFileSimple(srcPath, dstPath string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		os.Remove(dstPath)
		return err
	}
	return nil
}

// Init initializes the database and returns the connection plus a flag
// indicating whether the DB file was newly created (true) or already existed (false).
func Init(dataSourceName string) (*sql.DB, bool, error) {
	_, statErr := os.Stat(dataSourceName)
	isNew := os.IsNotExist(statErr)
	return initDB(dataSourceName, isNew)
}

func initDB(dataSourceName string, isNew bool) (*sql.DB, bool, error) {
	db, err := sql.Open("sqlite", dataSourceName)
	if err != nil {
		return nil, false, err
	}

	if err := db.Ping(); err != nil {
		return nil, false, err
	}

	// Enable foreign key enforcement
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, false, err
	}

	createTables := `
	CREATE TABLE IF NOT EXISTS customers (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		customer_number TEXT NOT NULL,
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

	CREATE TABLE IF NOT EXISTS quotes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		quote_number TEXT NOT NULL,
		date TEXT NOT NULL,
		sender_name TEXT NOT NULL,
		sender_address TEXT NOT NULL,
		recipient_name TEXT NOT NULL,
		recipient_address TEXT NOT NULL,
		tax_rate REAL DEFAULT 19.0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		status TEXT DEFAULT 'Entwurf' CHECK(status IN ('Entwurf', 'Verschickt', 'Angenommen', 'Abgelehnt', 'Umgewandelt')),
		is_small_business BOOLEAN DEFAULT 0,
		customer_id INTEGER REFERENCES customers(id) ON DELETE SET NULL
	);

	CREATE TABLE IF NOT EXISTS quote_items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		quote_id INTEGER NOT NULL REFERENCES quotes(id) ON DELETE CASCADE,
		description TEXT NOT NULL,
		quantity INTEGER NOT NULL,
		price_per_unit REAL NOT NULL,
		product_id INTEGER REFERENCES products(id) ON DELETE SET NULL
	);

	CREATE TABLE IF NOT EXISTS credit_notes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		credit_note_number TEXT NOT NULL,
		date TEXT NOT NULL,
		sender_name TEXT NOT NULL,
		sender_address TEXT NOT NULL,
		recipient_name TEXT NOT NULL,
		recipient_address TEXT NOT NULL,
		tax_rate REAL DEFAULT 19.0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		status TEXT DEFAULT 'Offen' CHECK(status IN ('Entwurf', 'Offen', 'Abgeschlossen')),
		is_small_business BOOLEAN DEFAULT 0,
		customer_id INTEGER REFERENCES customers(id) ON DELETE SET NULL,
		invoice_id INTEGER REFERENCES invoices(id) ON DELETE SET NULL
	);

	CREATE TABLE IF NOT EXISTS credit_note_items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		credit_note_id INTEGER NOT NULL REFERENCES credit_notes(id) ON DELETE CASCADE,
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
		slog.Error("Error creating tables", "error", err)
		return nil, false, err
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
		"ALTER TABLE customers ADD COLUMN customer_number TEXT",
	}

	for _, m := range migrations {
		_, _ = db.Exec(m)
	}

	// Fix missing customer numbers for existing customers
	_, _ = db.Exec("UPDATE customers SET customer_number = 'KD-' || printf('%04d', id) WHERE customer_number IS NULL OR customer_number = ''")

	// Migrate existing category text values into expense_categories table
	migrateCategories(db)

	return db, isNew, nil
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
