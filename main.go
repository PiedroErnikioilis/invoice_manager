package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"din-invoice/db"
	"din-invoice/handlers"
	"din-invoice/models"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Println("Press Enter to exit...")
		bufio.NewReader(os.Stdin).ReadBytes('\n')
		os.Exit(1)
	}
}

func run() error {
	// Setup structured logging
	var handler slog.Handler
	logLevel := slog.LevelInfo
	if os.Getenv("DEBUG") == "1" {
		logLevel = slog.LevelDebug
	}

	// Always log to file, and also stdout
	logFile, err := os.OpenFile("app.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open log file: %v\n", err)
	} else {
		defer logFile.Close()
	}

	var logWriter io.Writer = os.Stdout
	if logFile != nil {
		logWriter = io.MultiWriter(os.Stdout, logFile)
	}

	opts := &slog.HandlerOptions{Level: logLevel}
	if os.Getenv("JSON_LOG") == "1" {
		handler = slog.NewJSONHandler(logWriter, opts)
	} else {
		handler = slog.NewTextHandler(logWriter, opts)
	}
	logger := slog.New(handler)
	slog.SetDefault(logger)

	// 1. Pre-Migration-Backup (bevor Schema-Änderungen laufen)
	dbPath := "invoices.db"
	if err := db.PreMigrationBackup(dbPath); err != nil {
		slog.Error("Pre-Migration-Backup fehlgeschlagen", "error", err)
	}

	// 2. Init DB (erstellt Tabellen und führt Migrationen aus)
	database, isNewDB, err := db.Init(dbPath)
	if err != nil {
		return fmt.Errorf("failed to init db: %w", err)
	}
	defer database.Close()
	store := models.NewStore(database)

	// Demo-Modus: Beispieldaten nur bei neuer DB erstellen
	if isNewDB && slices.Contains(os.Args[1:], "--demo") {
		if err := store.SeedDemoData(); err != nil {
			slog.Error("Demo-Daten Fehler", "error", err)
		}
	}
	invoiceHandler := handlers.NewInvoiceHandler(store)
	settingsHandler := handlers.NewSettingsHandler(store)
	productHandler := handlers.NewProductHandler(store)
	customerHandler := handlers.NewCustomerHandler(store)
	statsHandler := handlers.NewStatsHandler(store)
	euerHandler := handlers.NewEuerHandler(store)
	quoteHandler := handlers.NewQuoteHandler(store)
	creditNoteHandler := handlers.NewCreditNoteHandler(store)
	backupHandler := handlers.NewBackupHandler(store, dbPath)

	// 2. Setup Router
	r := chi.NewRouter()
	r.Use(handlers.Logger(logger))
	r.Use(middleware.Recoverer)

	r.Get("/", invoiceHandler.List)
	r.Get("/invoices/new", invoiceHandler.New)
	r.Post("/invoices", invoiceHandler.Create)
	r.Get("/invoices/{id}", invoiceHandler.View)
	r.Get("/invoices/{id}/pdf", invoiceHandler.DownloadPDF)
	r.Get("/invoices/{id}/edit", invoiceHandler.Edit)
	r.Post("/invoices/{id}", invoiceHandler.Update)
	r.Post("/invoices/{id}/cancel", invoiceHandler.Cancel)

	r.Get("/quotes", quoteHandler.List)
	r.Get("/quotes/new", quoteHandler.New)
	r.Post("/quotes", quoteHandler.Create)
	r.Get("/quotes/{id}/edit", quoteHandler.Edit)
	r.Post("/quotes/{id}", quoteHandler.Update)
	r.Post("/quotes/{id}/convert", quoteHandler.ConvertToInvoice)

	r.Get("/credit-notes", creditNoteHandler.List)
	r.Get("/credit-notes/new", creditNoteHandler.NewFromInvoice)
	r.Post("/credit-notes", creditNoteHandler.Create)

	r.Get("/products", productHandler.List)
	r.Get("/products/new", productHandler.New)
	r.Post("/products", productHandler.Create)
	r.Get("/products/inventory/pdf", productHandler.DownloadInventoryPDF)
	r.Get("/products/{id}/edit", productHandler.Edit)
	r.Post("/products/{id}", productHandler.Update)
	r.Post("/products/{id}/delete", productHandler.Delete)
	r.Post("/products/{id}/stock/add", productHandler.AddStock)
	r.Post("/products/{id}/stock/remove", productHandler.RemoveStock)

	r.Get("/customers", customerHandler.List)
	r.Get("/customers/new", customerHandler.New)
	r.Post("/customers", customerHandler.Create)
	r.Get("/customers/{id}/edit", customerHandler.Edit)
	r.Post("/customers/{id}", customerHandler.Update)
	r.Post("/customers/{id}/delete", customerHandler.Delete)

	r.Get("/statistics", statsHandler.View)

	r.Get("/euer", euerHandler.View)
	r.Get("/euer/pdf", euerHandler.DownloadPDF)
	r.Get("/euer/csv", euerHandler.DownloadCSV)
	r.Get("/euer/recurring", euerHandler.ListRecurring)
	r.Get("/euer/recurring/new", euerHandler.NewRecurring)
	r.Post("/euer/recurring", euerHandler.CreateRecurring)
	r.Post("/euer/recurring/{id}/delete", euerHandler.DeleteRecurring)
	r.Get("/expenses/new", euerHandler.NewExpense)
	r.Post("/euer/expenses", euerHandler.CreateExpense)
	r.Get("/euer/expenses/{id}/edit", euerHandler.EditExpense)
	r.Post("/euer/expenses/{id}", euerHandler.UpdateExpense)
	r.Get("/euer/expenses/{id}/receipt", euerHandler.ServeReceipt)
	r.Post("/euer/expenses/{id}/delete", euerHandler.DeleteExpense)

	r.Get("/settings", settingsHandler.View)
	r.Post("/settings", settingsHandler.Save)

	r.Get("/backups", backupHandler.List)
	r.Post("/backups/create", backupHandler.Create)
	r.Get("/backups/{filename}/download", backupHandler.Download)
	r.Post("/backups/{filename}/delete", backupHandler.Delete)
	r.Post("/backups/{filename}/restore", backupHandler.Restore)

	// Static Files
	workDir, _ := os.Getwd()
	filesDir := http.Dir(filepath.Join(workDir, "uploads"))
	FileServer(r, "/uploads", filesDir)

	// 3. Start Server
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	slog.Info("Server starting", "url", "http://localhost:"+port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		return fmt.Errorf("server error: %w", err)
	}
	return nil
}

func FileServer(r chi.Router, path string, root http.FileSystem) {
	if strings.ContainsAny(path, "{}*") {
		panic("FileServer does not permit any URL parameters.")
	}

	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", 301).ServeHTTP)
		path += "/"
	}
	path += "*"

	r.Get(path, func(w http.ResponseWriter, r *http.Request) {
		rctx := chi.RouteContext(r.Context())
		pathPrefix := strings.TrimSuffix(rctx.RoutePattern(), "/*")
		fs := http.StripPrefix(pathPrefix, http.FileServer(root))
		fs.ServeHTTP(w, r)
	})
}
