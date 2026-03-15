package handlers

import (
	"din-invoice/models"
	"din-invoice/services"
	"din-invoice/views"
	"encoding/base64"
	"fmt"
	"io"
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
	// 0. Process recurring expenses before viewing
	h.Store.ProcessRecurringExpenses()

	// Default to current year
	year := time.Now().Year()
	if y := r.URL.Query().Get("year"); y != "" {
		if parsed, err := strconv.Atoi(y); err == nil && parsed > 0 {
			year = parsed
		}
	}

	stats, err := h.Store.GetEuerStats(year)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	years, _ := h.Store.GetAvailableYears()

	views.EuerDashboard(stats, years).Render(r.Context(), w)
}

func (h *EuerHandler) ListRecurring(w http.ResponseWriter, r *http.Request) {
	list, err := h.Store.ListRecurringExpenses()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	views.RecurringExpenseList(list).Render(r.Context(), w)
}

func (h *EuerHandler) NewRecurring(w http.ResponseWriter, r *http.Request) {
	categories, _ := h.Store.ListExpenseCategories()
	views.RecurringExpenseForm(categories).Render(r.Context(), w)
}

func (h *EuerHandler) CreateRecurring(w http.ResponseWriter, r *http.Request) {
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

	categoryName := strings.TrimSpace(r.FormValue("category"))
	if categoryName != "" {
		catID, err := h.Store.CreateExpenseCategory(categoryName)
		if err == nil {
			re.CategoryID = &catID
		}
	}

	if re.StartDate == "" {
		re.StartDate = time.Now().Format("2006-01-02")
	}

	_, err := h.Store.CreateRecurringExpense(re)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/euer/recurring", http.StatusSeeOther)
}

func (h *EuerHandler) DeleteRecurring(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	err = h.Store.DeleteRecurringExpense(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/euer/recurring", http.StatusSeeOther)
}

func (h *EuerHandler) NewExpense(w http.ResponseWriter, r *http.Request) {
	products, err := h.Store.ListProducts()
	if err != nil {
		products = []models.Product{}
	}
	categories, err := h.Store.ListExpenseCategories()
	if err != nil {
		categories = []models.ExpenseCategory{}
	}
	views.ExpenseForm(products, categories, nil).Render(r.Context(), w)
}

func (h *EuerHandler) EditExpense(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	expense, err := h.Store.GetExpense(id)
	if err != nil {
		http.Error(w, "Expense not found", http.StatusNotFound)
		return
	}

	products, err := h.Store.ListProducts()
	if err != nil {
		products = []models.Product{}
	}
	categories, err := h.Store.ListExpenseCategories()
	if err != nil {
		categories = []models.ExpenseCategory{}
	}
	views.ExpenseForm(products, categories, &expense).Render(r.Context(), w)
}

func (h *EuerHandler) UpdateExpense(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10MB
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

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

	// Resolve or create category
	categoryName := strings.TrimSpace(r.FormValue("category"))
	if categoryName != "" {
		catID, err := h.Store.CreateExpenseCategory(categoryName)
		if err == nil {
			expense.CategoryID = &catID
		}
	}

	// Handle Receipt Upload (Optional for update)
	file, handler, err := r.FormFile("receipt")
	if err == nil {
		defer file.Close()

		// Read file content
		fileBytes, err := io.ReadAll(file)
		if err == nil {
			// Encode to Base64
			expense.ReceiptData = base64.StdEncoding.EncodeToString(fileBytes)
			expense.ReceiptPath = handler.Filename
		}
	}

	err = h.Store.UpdateExpense(expense)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/euer", http.StatusSeeOther)
}

func (h *EuerHandler) CreateExpense(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10MB
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

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

	// Resolve or create category
	categoryName := strings.TrimSpace(r.FormValue("category"))
	if categoryName != "" {
		catID, err := h.Store.CreateExpenseCategory(categoryName)
		if err == nil {
			expense.CategoryID = &catID
		}
	}

	if expense.Date == "" {
		expense.Date = time.Now().Format("2006-01-02")
	}

	// Handle Receipt Upload
	file, handler, err := r.FormFile("receipt")
	if err == nil {
		defer file.Close()

		// Read file content
		fileBytes, err := io.ReadAll(file)
		if err == nil {
			// Encode to Base64
			expense.ReceiptData = base64.StdEncoding.EncodeToString(fileBytes)
			expense.ReceiptPath = handler.Filename // Store original filename for extension/mime
		}
	}

	_, err = h.Store.CreateExpense(expense)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Handle Inventory Link
	if r.FormValue("update_inventory") == "on" {
		productID, _ := strconv.Atoi(r.FormValue("product_id"))
		quantity, _ := strconv.Atoi(r.FormValue("quantity"))

		if productID > 0 && quantity > 0 {
			// Record stock addition (Purchase)
			h.Store.RecordStockMovement(productID, quantity, "PURCHASE", "Einkauf: "+expense.Description)
		}
	}

	http.Redirect(w, r, "/euer", http.StatusSeeOther)
}

func (h *EuerHandler) DownloadPDF(w http.ResponseWriter, r *http.Request) {
	year := time.Now().Year()
	if y := r.URL.Query().Get("year"); y != "" {
		if parsed, err := strconv.Atoi(y); err == nil && parsed > 0 {
			year = parsed
		}
	}

	stats, err := h.Store.GetEuerStats(year)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	settings, err := h.Store.GetAppSettings()
	if err != nil {
		http.Error(w, "Could not load settings", http.StatusInternalServerError)
		return
	}

	path, err := services.GenerateEuerPDFHTML(stats, &settings)
	if err != nil {
		http.Error(w, "Failed to generate PDF: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	filename := models.FormatDocumentNumber(settings.EuerFilenameSchema, 0) + ".pdf"
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%s", filename))
	http.ServeFile(w, r, path)
}

func (h *EuerHandler) DownloadCSV(w http.ResponseWriter, r *http.Request) {
	year := time.Now().Year()
	if y := r.URL.Query().Get("year"); y != "" {
		if parsed, err := strconv.Atoi(y); err == nil && parsed > 0 {
			year = parsed
		}
	}

	stats, err := h.Store.GetEuerStats(year)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	settings, _ := h.Store.GetAppSettings()
	filename := models.FormatDocumentNumber(settings.EuerFilenameSchema, 0) + ".csv"

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

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
}

func (h *EuerHandler) ServeReceipt(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	filename, data, err := h.Store.GetExpenseReceipt(id)
	if err != nil {
		http.Error(w, "Receipt not found", http.StatusNotFound)
		return
	}

	// Serve from DB (Base64)
	if data != "" {
		decoded, err := base64.StdEncoding.DecodeString(data)
		if err != nil {
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
		return
	}

	// Fallback to filesystem (Legacy)
	if filename == "" {
		http.Error(w, "No receipt", http.StatusNotFound)
		return
	}

	http.ServeFile(w, r, filename)
}

func (h *EuerHandler) DeleteExpense(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := h.Store.DeleteExpense(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/euer", http.StatusSeeOther)
}
