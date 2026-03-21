package handlers

import (
	"din-invoice/models"
	"din-invoice/services"
	"din-invoice/views"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

type QuoteHandler struct {
	Store *models.Store
}

func NewQuoteHandler(store *models.Store) *QuoteHandler {
	return &QuoteHandler{Store: store}
}

func (h *QuoteHandler) List(w http.ResponseWriter, r *http.Request) {
	slog.Debug("Processing List quotes request", "method", r.Method)
	quotes, err := h.Store.ListQuotes()
	if err != nil {
		slog.Error("Failed to list quotes", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	slog.Debug("Successfully listed quotes", "count", len(quotes))
	views.QuoteList(quotes).Render(r.Context(), w)
}

func (h *QuoteHandler) New(w http.ResponseWriter, r *http.Request) {
	slog.Debug("Processing New quote request", "method", r.Method)
	customers, _ := h.Store.ListCustomers()
	products, _ := h.Store.ListProducts()
	settings, _ := h.Store.GetAppSettings()

	quoteNum := models.FormatDocumentNumber(settings.QuoteNumberSchema, settings.NextQuoteNumber)

	quote := &models.Quote{
		QuoteNumber:   quoteNum,
		Date:          time.Now().Format("2006-01-02"),
		SenderName:    settings.SenderName,
		SenderAddress: settings.SenderAddress,
		TaxRate:       19.0,
	}

	slog.Debug("Successfully prepared new quote form", "quote_number", quoteNum)
	views.QuoteForm(quote, customers, products).Render(r.Context(), w)
}

func (h *QuoteHandler) Create(w http.ResponseWriter, r *http.Request) {
	slog.Debug("Processing Create quote request", "method", r.Method)
	quote, err := h.parseForm(r)
	if err != nil {
		slog.Error("Failed to parse quote form", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	slog.Debug("Successfully parsed quote form", "quote_number", quote.QuoteNumber)

	slog.Info("Creating quote", "quote_number", quote.QuoteNumber)

	// Increment quote number if it matches the auto-generated one
	settings, _ := h.Store.GetAppSettings()
	expectedNum := models.FormatDocumentNumber(settings.QuoteNumberSchema, settings.NextQuoteNumber)
	if quote.QuoteNumber == expectedNum {
		slog.Debug("Incrementing next quote number", "quote_number", quote.QuoteNumber)
		h.Store.IncrementNextQuoteNumber()
	}

	id, err := h.Store.CreateQuote(quote)
	if err != nil {
		slog.Error("Failed to create quote", "quote_number", quote.QuoteNumber, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	slog.Info("Quote created successfully", "id", id, "quote_number", quote.QuoteNumber)
	http.Redirect(w, r, "/quotes", http.StatusSeeOther)
}

func (h *QuoteHandler) Edit(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		slog.Error("Failed to parse quote ID for edit", "id", idStr, "error", err)
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	slog.Debug("Processing Edit quote request", "id", id, "method", r.Method)

	slog.Debug("Fetching quote for edit", "id", id)
	quote, err := h.Store.GetQuote(id)
	if err != nil {
		slog.Error("Quote not found for edit", "id", id, "error", err)
		http.Error(w, "Quote not found", http.StatusNotFound)
		return
	}

	customers, _ := h.Store.ListCustomers()
	products, _ := h.Store.ListProducts()

	slog.Debug("Successfully fetched quote and lists for edit", "id", id)
	views.QuoteForm(quote, customers, products).Render(r.Context(), w)
}

func (h *QuoteHandler) View(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		slog.Error("Failed to parse quote ID for view", "id", idStr, "error", err)
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	slog.Debug("Processing View quote request", "id", id, "method", r.Method)

	slog.Debug("Fetching quote for view", "id", id)
	quote, err := h.Store.GetQuote(id)
	if err != nil {
		slog.Error("Quote not found for view", "id", id, "error", err)
		http.Error(w, "Quote not found", http.StatusNotFound)
		return
	}

	settings, _ := h.Store.GetAppSettings()
	slog.Debug("Successfully fetched quote and settings for view", "id", id)
	views.QuoteView(quote, settings).Render(r.Context(), w)
}

func (h *QuoteHandler) Update(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		slog.Error("Failed to parse quote ID for update", "id", idStr, "error", err)
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	slog.Debug("Processing Update quote request", "id", id, "method", r.Method)

	quote, err := h.parseForm(r)
	if err != nil {
		slog.Error("Failed to parse quote update form", "id", id, "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	quote.ID = id
	slog.Debug("Successfully parsed quote update form", "id", id, "quote_number", quote.QuoteNumber)

	slog.Info("Updating quote", "id", id, "quote_number", quote.QuoteNumber)
	err = h.Store.UpdateQuote(quote)
	if err != nil {
		slog.Error("Failed to update quote", "id", id, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	slog.Info("Quote updated successfully", "id", id)
	http.Redirect(w, r, "/quotes", http.StatusSeeOther)
}

func (h *QuoteHandler) ConvertToInvoice(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		slog.Error("Failed to parse quote ID for conversion", "id", idStr, "error", err)
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	slog.Debug("Processing ConvertToInvoice request", "id", id, "method", r.Method)

	slog.Info("Converting quote to invoice", "quote_id", id)
	quote, err := h.Store.GetQuote(id)
	if err != nil {
		slog.Error("Quote not found for conversion", "id", id, "error", err)
		http.Error(w, "Quote not found", http.StatusNotFound)
		return
	}

	// 1. Create Invoice from Quote
	invoice := &models.Invoice{
		InvoiceNumber:    "RE-" + quote.QuoteNumber, // Prefix or similar
		Date:             time.Now().Format("2006-01-02"),
		SenderName:       quote.SenderName,
		SenderAddress:    quote.SenderAddress,
		RecipientName:    quote.RecipientName,
		RecipientAddress: quote.RecipientAddress,
		TaxRate:          quote.TaxRate,
		Status:           "Offen",
		IsSmallBusiness:  quote.IsSmallBusiness,
		CustomerID:       quote.CustomerID,
	}

	for _, item := range quote.Items {
		invoice.Items = append(invoice.Items, models.InvoiceItem{
			Description:  item.Description,
			Quantity:     item.Quantity,
			PricePerUnit: item.PricePerUnit,
			ProductID:    item.ProductID,
		})
	}

	slog.Debug("Creating invoice from quote", "quote_id", id, "invoice_number", invoice.InvoiceNumber)
	invID, err := h.Store.CreateInvoice(invoice)
	if err != nil {
		slog.Error("Failed to create invoice from quote", "quote_id", id, "error", err)
		http.Error(w, "Failed to create invoice: "+err.Error(), http.StatusInternalServerError)
		return
	}

	slog.Info("Invoice created from quote", "quote_id", id, "invoice_id", invID, "invoice_number", invoice.InvoiceNumber)

	// 2. Update Quote Status
	slog.Debug("Updating quote status to 'Umgewandelt'", "quote_id", id)
	quote.Status = "Umgewandelt"
	if err := h.Store.UpdateQuote(quote); err != nil {
		slog.Error("Failed to update quote status after conversion", "quote_id", id, "error", err)
	} else {
		slog.Debug("Quote status updated to 'Umgewandelt'", "quote_id", id)
	}

	// 3. Redirect to new Invoice
	http.Redirect(w, r, fmt.Sprintf("/invoices/%d", invID), http.StatusSeeOther)
}

func (h *QuoteHandler) parseForm(r *http.Request) (*models.Quote, error) {
	slog.Debug("Parsing quote form", "method", r.Method)
	if err := r.ParseForm(); err != nil {
		slog.Error("Failed to parse quote form", "error", err)
		return nil, err
	}

	taxRate := parseDecimal(r.FormValue("tax_rate"))
	customerIDStr := r.FormValue("customer_id")
	var customerID *int
	if customerIDStr != "" {
		id, _ := strconv.Atoi(customerIDStr)
		customerID = &id
	}

	quote := &models.Quote{
		QuoteNumber:      r.FormValue("quote_number"),
		Date:             r.FormValue("date"),
		SenderName:       r.FormValue("sender_name"),
		SenderAddress:    r.FormValue("sender_address"),
		RecipientName:    r.FormValue("recipient_name"),
		RecipientAddress: r.FormValue("recipient_address"),
		TaxRate:          taxRate,
		Status:           r.FormValue("status"),
		IsSmallBusiness:  r.FormValue("is_small_business") == "true",
		CustomerID:       customerID,
	}

	// Parse items
	descriptions := r.Form["item_description[]"]
	quantities := r.Form["item_quantity[]"]
	prices := r.Form["item_price[]"]
	productIDs := r.Form["item_product_id[]"]

	slog.Debug("Parsing quote items", "count", len(descriptions))
	for i := range descriptions {
		if descriptions[i] == "" {
			continue
		}
		qty, _ := strconv.Atoi(quantities[i])
		price := parseDecimal(prices[i])
		var pID *int
		if i < len(productIDs) && productIDs[i] != "" {
			id, _ := strconv.Atoi(productIDs[i])
			pID = &id
		}

		quote.Items = append(quote.Items, models.QuoteItem{
			Description:  descriptions[i],
			Quantity:     qty,
			PricePerUnit: price,
			ProductID:    pID,
		})
	}

	slog.Debug("Successfully parsed quote form", "quote_number", quote.QuoteNumber, "items_count", len(quote.Items))
	return quote, nil
}

func (h *QuoteHandler) DownloadPDF(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		slog.Error("Failed to parse quote ID for PDF download", "id", idStr, "error", err)
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	slog.Debug("Processing DownloadPDF quote request", "id", id, "method", r.Method)

	force := r.URL.Query().Get("force") == "1"

	slog.Debug("Fetching quote for PDF", "id", id)
	quote, err := h.Store.GetQuote(id)
	if err != nil {
		slog.Error("Quote not found for PDF", "id", id, "error", err)
		http.Error(w, "Quote not found", http.StatusNotFound)
		return
	}

	settings, err := h.Store.GetAppSettings()
	if err != nil {
		slog.Error("Failed to load settings for quote PDF", "error", err)
		http.Error(w, "Could not load settings", http.StatusInternalServerError)
		return
	}

	path := services.GetQuotePDFPath(quote, &settings)
	
	// Smart Check: If file exists and state is final, don't regenerate unless forced
	isFinal := quote.Status == "Angenommen" || quote.Status == "Abgelehnt" || quote.Status == "Umgewandelt"
	if !force && isFinal {
		if _, err := os.Stat(path); err == nil {
			slog.Debug("Serving existing quote PDF", "path", path, "status", quote.Status)
			w.Header().Set("Content-Type", "application/pdf")
			w.Header().Set("Content-Disposition", "inline; filename="+filepath.Base(path))
			http.ServeFile(w, r, path)
			return
		}
	}

	slog.Info("Generating fresh quote PDF", "id", id, "force", force, "is_final", isFinal)
	path, err = services.GenerateQuotePDFHTML(quote, &settings)
	if err != nil {
		slog.Error("Failed to generate quote PDF", "id", id, "error", err)
		http.Error(w, "Failed to generate PDF: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "inline; filename="+filepath.Base(path))
	slog.Debug("Serving freshly generated quote PDF", "path", path)
	http.ServeFile(w, r, path)
}
