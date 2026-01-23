package main

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
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
	// 1. Init DB
	database, err := db.Init("invoices.db")
	if err != nil {
		return fmt.Errorf("failed to init db: %w", err)
	}
	defer database.Close()

	store := models.NewStore(database)
	invoiceHandler := handlers.NewInvoiceHandler(store)
	settingsHandler := handlers.NewSettingsHandler(store)
	productHandler := handlers.NewProductHandler(store)
	customerHandler := handlers.NewCustomerHandler(store)
	statsHandler := handlers.NewStatsHandler(store)
	euerHandler := handlers.NewEuerHandler(store)

	// 2. Setup Router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/", invoiceHandler.List)
	r.Get("/invoices/new", invoiceHandler.New)
	r.Post("/invoices", invoiceHandler.Create)
	r.Get("/invoices/{id}", invoiceHandler.View)
	r.Get("/invoices/{id}/pdf", invoiceHandler.DownloadPDF)
	r.Get("/invoices/{id}/edit", invoiceHandler.Edit)
	r.Post("/invoices/{id}", invoiceHandler.Update)

	r.Get("/products", productHandler.List)
	r.Get("/products/new", productHandler.New)
	r.Post("/products", productHandler.Create)
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
	r.Get("/expenses/new", euerHandler.NewExpense)
	r.Post("/euer/expenses", euerHandler.CreateExpense)
	r.Post("/euer/expenses/{id}/delete", euerHandler.DeleteExpense)

	r.Get("/settings", settingsHandler.View)
	r.Post("/settings", settingsHandler.Save)

	// Static Files
	workDir, _ := os.Getwd()
	filesDir := http.Dir(filepath.Join(workDir, "uploads"))
	FileServer(r, "/uploads", filesDir)

	// 3. Start Server
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	fmt.Printf("Server starting on http://localhost:%s\n", port)
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
