package handlers

import (
	"din-invoice/models"
	"din-invoice/services"
	"din-invoice/views"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

type EuerHandler struct {
	Store *models.Store
}

func NewEuerHandler(store *models.Store) *EuerHandler {
	return &EuerHandler{Store: store}
}

func (h *EuerHandler) View(w http.ResponseWriter, r *http.Request) {
	slog.Debug("Processing View EÜR request", "method", r.Method)
	// 0. Process recurring expenses before viewing
	h.Store.ProcessRecurringExpenses()

	// Default to current year
	year := time.Now().Year()
	if y := r.URL.Query().Get("year"); y != "" {
		if parsed, err := strconv.Atoi(y); err == nil && parsed > 0 {
			year = parsed
		}
	}

	slog.Debug("Fetching EÜR stats from store", "year", year)
	stats, err := h.Store.GetEuerStats(year)
	if err != nil {
		slog.Error("Failed to get EÜR stats", "year", year, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	slog.Debug("Successfully fetched EÜR stats", "year", year)

	years, _ := h.Store.GetAvailableYears()

	views.EuerDashboard(stats, years).Render(r.Context(), w)
}

func (h *EuerHandler) ListRecurring(w http.ResponseWriter, r *http.Request) {
	slog.Debug("Processing ListRecurring request", "method", r.Method)
	list, err := h.Store.ListRecurringExpenses()
	if err != nil {
		slog.Error("Failed to list recurring expenses", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	slog.Debug("Successfully listed recurring expenses", "count", len(list))
	views.RecurringExpenseList(list).Render(r.Context(), w)
}

func (h *EuerHandler) NewRecurring(w http.ResponseWriter, r *http.Request) {
	slog.Debug("Processing NewRecurring request", "method", r.Method)
	categories, _ := h.Store.ListExpenseCategories()
	slog.Debug("Successfully listed expense categories for new recurring expense", "count", len(categories))
	views.RecurringExpenseForm(categories).Render(r.Context(), w)
}

func (h *EuerHandler) CreateRecurring(w http.ResponseWriter, r *http.Request) {
	slog.Debug("Processing CreateRecurring request", "method", r.Method)
	amount := parseDecimal(r.FormValue("amount"))
	taxRate := parseDecimal(r.FormValue("tax_rate"))
	if taxRate == 0 && r.FormValue("tax_rate") == "" {
		taxRate = 19.0
	}

	re := models.RecurringExpense{
		Description: r.FormValue("description"),
		Amount:      amount,
		TaxRate:     taxRate,
		Interval:    r.FormValue("interval"),
		StartDate:   r.FormValue("start_date"),
		IsActive:    true,
	}
	slog.Debug("Successfully parsed recurring expense form", "description", re.Description, "amount", re.Amount)

	slog.Info("Creating recurring expense", "description", re.Description, "amount", re.Amount)

	categoryName := strings.TrimSpace(r.FormValue("category"))
	if categoryName != "" {
		slog.Debug("Resolving expense category", "category", categoryName)
		catID, err := h.Store.CreateExpenseCategory(categoryName)
		if err == nil {
			re.CategoryID = &catID
			slog.Debug("Resolved expense category", "category", categoryName, "id", catID)
		} else {
			slog.Error("Failed to resolve expense category", "category", categoryName, "error", err)
		}
	}

	if re.StartDate == "" {
		re.StartDate = time.Now().Format("2006-01-02")
	}

	id, err := h.Store.CreateRecurringExpense(re)
	if err != nil {
		slog.Error("Failed to create recurring expense", "description", re.Description, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	slog.Info("Recurring expense created successfully", "id", id)
	http.Redirect(w, r, "/euer/recurring", http.StatusSeeOther)
}

func (h *EuerHandler) DeleteRecurring(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		slog.Error("Failed to parse recurring expense ID for deletion", "id", idStr, "error", err)
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	slog.Debug("Processing DeleteRecurring request", "id", id, "method", r.Method)

	slog.Info("Deleting recurring expense", "id", id)
	err = h.Store.DeleteRecurringExpense(id)
	if err != nil {
		slog.Error("Failed to delete recurring expense", "id", id, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	slog.Info("Recurring expense deleted successfully", "id", id)
	http.Redirect(w, r, "/euer/recurring", http.StatusSeeOther)
}

func (h *EuerHandler) NewExpense(w http.ResponseWriter, r *http.Request) {
	slog.Debug("Processing NewExpense request", "method", r.Method)
	products, err := h.Store.ListProducts()
	if err != nil {
		slog.Error("Failed to list products for new expense form", "error", err)
		products = []models.Product{}
	}
	categories, err := h.Store.ListExpenseCategories()
	if err != nil {
		slog.Error("Failed to list categories for new expense form", "error", err)
		categories = []models.ExpenseCategory{}
	}
	slog.Debug("Successfully listed products and categories for new expense", "products_count", len(products), "categories_count", len(categories))
	views.ExpenseForm(products, categories, nil).Render(r.Context(), w)
}

func (h *EuerHandler) EditExpense(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		slog.Error("Failed to parse expense ID for edit", "id", idStr, "error", err)
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	slog.Debug("Processing EditExpense request", "id", id, "method", r.Method)

	slog.Debug("Fetching expense for edit", "id", id)
	expense, err := h.Store.GetExpense(id)
	if err != nil {
		slog.Error("Expense not found for edit", "id", id, "error", err)
		http.Error(w, "Expense not found", http.StatusNotFound)
		return
	}

	products, err := h.Store.ListProducts()
	if err != nil {
		slog.Error("Failed to list products for edit expense form", "error", err)
		products = []models.Product{}
	}
	categories, err := h.Store.ListExpenseCategories()
	if err != nil {
		slog.Error("Failed to list categories for edit expense form", "error", err)
		categories = []models.ExpenseCategory{}
	}
	slog.Debug("Successfully fetched expense and lists for edit", "id", id)
	views.ExpenseForm(products, categories, &expense).Render(r.Context(), w)
}

func (h *EuerHandler) UpdateExpense(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		slog.Error("Failed to parse expense ID for update", "id", idStr, "error", err)
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	slog.Debug("Processing UpdateExpense request", "id", id, "method", r.Method)

	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10MB
		slog.Error("Failed to parse multipart form for expense update", "id", id, "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	slog.Debug("Successfully parsed multipart form for expense update", "id", id)

	amount := parseDecimal(r.FormValue("amount"))
	taxRate := parseDecimal(r.FormValue("tax_rate"))
	if taxRate == 0 && r.FormValue("tax_rate") == "" {
		taxRate = 19.0
	}

	expense := models.Expense{
		ID:          id,
		Description: r.FormValue("description"),
		Amount:      amount,
		TaxRate:     taxRate,
		Date:        r.FormValue("date"),
	}

	slog.Info("Updating expense", "id", id, "description", expense.Description)

	// Resolve or create category
	categoryName := strings.TrimSpace(r.FormValue("category"))
	if categoryName != "" {
		slog.Debug("Resolving expense category", "id", id, "category", categoryName)
		catID, err := h.Store.CreateExpenseCategory(categoryName)
		if err == nil {
			expense.CategoryID = &catID
			slog.Debug("Resolved expense category", "id", id, "category", categoryName, "cat_id", catID)
		} else {
			slog.Error("Failed to resolve expense category", "id", id, "category", categoryName, "error", err)
		}
	}

	// Handle Receipt Upload (Optional for update)
	file, handler, err := r.FormFile("receipt")
	if err == nil {
		defer file.Close()
		slog.Debug("Uploading new receipt for expense", "id", id, "filename", handler.Filename)

		// Read file content
		fileBytes, err := io.ReadAll(file)
		if err == nil {
			// Encode to Base64
			expense.ReceiptData = base64.StdEncoding.EncodeToString(fileBytes)
			expense.ReceiptPath = handler.Filename
			slog.Debug("Successfully read and encoded new receipt", "id", id, "filename", handler.Filename)
		} else {
			slog.Error("Failed to read receipt file", "id", id, "error", err)
		}
	}

	err = h.Store.UpdateExpense(expense)
	if err != nil {
		slog.Error("Failed to update expense", "id", id, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	slog.Info("Expense updated successfully", "id", id)
	http.Redirect(w, r, "/euer", http.StatusSeeOther)
}

func (h *EuerHandler) CreateExpense(w http.ResponseWriter, r *http.Request) {
	slog.Debug("Processing CreateExpense request", "method", r.Method)
	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10MB
		slog.Error("Failed to parse multipart form for expense creation", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	slog.Debug("Successfully parsed multipart form for expense creation")

	amount := parseDecimal(r.FormValue("amount"))
	taxRate := parseDecimal(r.FormValue("tax_rate"))
	if taxRate == 0 && r.FormValue("tax_rate") == "" {
		taxRate = 19.0
	}

	expense := models.Expense{
		Description: r.FormValue("description"),
		Amount:      amount,
		TaxRate:     taxRate,
		Date:        r.FormValue("date"),
	}

	slog.Info("Creating expense", "description", expense.Description, "amount", expense.Amount)

	// Resolve or create category
	categoryName := strings.TrimSpace(r.FormValue("category"))
	if categoryName != "" {
		slog.Debug("Resolving expense category", "category", categoryName)
		catID, err := h.Store.CreateExpenseCategory(categoryName)
		if err == nil {
			expense.CategoryID = &catID
			slog.Debug("Resolved expense category", "category", categoryName, "cat_id", catID)
		} else {
			slog.Error("Failed to resolve expense category", "category", categoryName, "error", err)
		}
	}

	if expense.Date == "" {
		expense.Date = time.Now().Format("2006-01-02")
	}

	// Handle Receipt Upload
	file, handler, err := r.FormFile("receipt")
	if err == nil {
		defer file.Close()
		slog.Debug("Uploading receipt for new expense", "filename", handler.Filename)

		// Read file content
		fileBytes, err := io.ReadAll(file)
		if err == nil {
			// Encode to Base64
			expense.ReceiptData = base64.StdEncoding.EncodeToString(fileBytes)
			expense.ReceiptPath = handler.Filename // Store original filename for extension/mime
			slog.Debug("Successfully read and encoded receipt", "filename", handler.Filename)
		} else {
			slog.Error("Failed to read receipt file", "error", err)
		}
	}

	id, err := h.Store.CreateExpense(expense)
	if err != nil {
		slog.Error("Failed to create expense", "description", expense.Description, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	slog.Info("Expense created successfully", "id", id)

	// Handle Inventory Link
	if r.FormValue("update_inventory") == "on" {
		productID, _ := strconv.Atoi(r.FormValue("product_id"))
		quantity, _ := strconv.Atoi(r.FormValue("quantity"))

		if productID > 0 && quantity > 0 {
			slog.Info("Recording inventory link for expense", "expense_id", id, "product_id", productID, "quantity", quantity)
			// Record stock addition (Purchase)
			h.Store.RecordStockMovement(productID, quantity, "PURCHASE", "Einkauf: "+expense.Description)
		}
	}

	http.Redirect(w, r, "/euer", http.StatusSeeOther)
}

func (h *EuerHandler) DownloadPDF(w http.ResponseWriter, r *http.Request) {
	slog.Debug("Processing DownloadPDF EÜR request", "method", r.Method)
	year := time.Now().Year()
	if y := r.URL.Query().Get("year"); y != "" {
		if parsed, err := strconv.Atoi(y); err == nil && parsed > 0 {
			year = parsed
		}
	}

	slog.Info("Generating EÜR PDF", "year", year)
	stats, err := h.Store.GetEuerStats(year)
	if err != nil {
		slog.Error("Failed to get EÜR stats for PDF", "year", year, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	settings, err := h.Store.GetAppSettings()
	if err != nil {
		slog.Error("Failed to load settings for EÜR PDF", "error", err)
		http.Error(w, "Could not load settings", http.StatusInternalServerError)
		return
	}

	path, err := services.GenerateEuerPDFHTML(stats, &settings)
	if err != nil {
		slog.Error("Failed to generate EÜR PDF", "year", year, "error", err)
		http.Error(w, "Failed to generate PDF: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	filename := models.FormatDocumentNumber(settings.EuerFilenameSchema, 0) + ".pdf"
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%s", filename))
	slog.Debug("Serving EÜR PDF", "path", path)
	http.ServeFile(w, r, path)
}

func (h *EuerHandler) DownloadCSV(w http.ResponseWriter, r *http.Request) {
	slog.Debug("Processing DownloadCSV EÜR request", "method", r.Method)
	year := time.Now().Year()
	if y := r.URL.Query().Get("year"); y != "" {
		if parsed, err := strconv.Atoi(y); err == nil && parsed > 0 {
			year = parsed
		}
	}

	slog.Info("Generating EÜR CSV", "year", year)
	stats, err := h.Store.GetEuerStats(year)
	if err != nil {
		slog.Error("Failed to get EÜR stats for CSV", "year", year, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	settings, _ := h.Store.GetAppSettings()
	filename := models.FormatDocumentNumber(settings.EuerFilenameSchema, 0) + ".csv"

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

	slog.Debug("Writing EÜR CSV data", "year", year)
	// German Excel uses semicolon as separator
	fmt.Fprintf(w, "Datum;Typ;Beleg-Nr;Beschreibung;Kategorie;Netto;USt%%;USt-Betrag;Brutto\n")

	// 1. Invoices (Income)
	for _, i := range stats.Invoices {
		fmt.Fprintf(w, "%s;Einnahme;%s;%s;%s;%.2f;%.1f;%.2f;%.2f\n",
			i.Date,
			i.InvoiceNumber,
			"Rechnung an "+i.RecipientName,
			"Umsatzerlöse",
			i.TotalNet(),
			i.TaxRate,
			i.TaxAmount(),
			i.TotalGross(),
		)
	}

	// 2. Expenses
	for _, e := range stats.Expenses {
		fmt.Fprintf(w, "%s;Ausgabe;%d;%s;%s;%.2f;%.1f;%.2f;%.2f\n",
			e.Date,
			e.ID,
			e.Description,
			e.CategoryName,
			e.Net(),
			e.TaxRate,
			e.Tax(),
			e.Amount,
		)
	}
	slog.Info("EÜR CSV generated successfully", "year", year)
}

func (h *EuerHandler) ServeReceipt(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		slog.Error("Failed to parse expense ID for receipt serving", "id", idStr, "error", err)
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	slog.Debug("Processing ServeReceipt request", "id", id, "method", r.Method)

	slog.Debug("Fetching expense receipt from store", "id", id)
	filename, data, err := h.Store.GetExpenseReceipt(id)
	if err != nil {
		slog.Error("Receipt not found", "id", id, "error", err)
		http.Error(w, "Receipt not found", http.StatusNotFound)
		return
	}

	// Serve from DB (Base64)
	if data != "" {
		slog.Debug("Serving receipt from database", "id", id, "filename", filename)
		decoded, err := base64.StdEncoding.DecodeString(data)
		if err != nil {
			slog.Error("Failed to decode receipt data", "id", id, "error", err)
			http.Error(w, "Error decoding receipt", http.StatusInternalServerError)
			return
		}

		// Detect content type
		ext := filepath.Ext(filename)
		mimeType := "application/octet-stream"
		switch strings.ToLower(ext) {
		case ".pdf":
			mimeType = "application/pdf"
		case ".png":
			mimeType = "image/png"
		case ".jpg", ".jpeg":
			mimeType = "image/jpeg"
		}
		w.Header().Set("Content-Type", mimeType)
		w.Write(decoded)
		slog.Info("Successfully served receipt from database", "id", id, "mime", mimeType)
		return
	}

	// Fallback to filesystem (Legacy)
	if filename == "" {
		slog.Error("No receipt path or data for expense", "id", id)
		http.Error(w, "No receipt", http.StatusNotFound)
		return
	}

	slog.Debug("Serving legacy receipt from filesystem", "id", id, "path", filename)
	http.ServeFile(w, r, filename)
}

func (h *EuerHandler) DeleteExpense(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		slog.Error("Failed to parse expense ID for deletion", "id", idStr, "error", err)
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	slog.Debug("Processing DeleteExpense request", "id", id, "method", r.Method)

	slog.Info("Deleting expense", "id", id)
	if err := h.Store.DeleteExpense(id); err != nil {
		slog.Error("Failed to delete expense", "id", id, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	slog.Info("Expense deleted successfully", "id", id)
	http.Redirect(w, r, "/euer", http.StatusSeeOther)
}
