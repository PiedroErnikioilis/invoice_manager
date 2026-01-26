package handlers

import (
	"din-invoice/models"
	"din-invoice/views"
	"encoding/base64"
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
	stats, err := h.Store.GetEuerStats()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	views.EuerDashboard(stats).Render(r.Context(), w)
}

func (h *EuerHandler) NewExpense(w http.ResponseWriter, r *http.Request) {
	products, err := h.Store.ListProducts()
	if err != nil {
		// Just log or empty list?
		products = []models.Product{}
	}
	views.ExpenseForm(products).Render(r.Context(), w)
}

func (h *EuerHandler) CreateExpense(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10MB
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	amount, _ := strconv.ParseFloat(r.FormValue("amount"), 64)

	expense := models.Expense{
		Description: r.FormValue("description"),
		Amount:      amount,
		Date:        r.FormValue("date"),
		Category:    r.FormValue("category"),
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
