package handlers

import (
	"din-invoice/models"
	"din-invoice/views"
	"fmt"
	"log/slog"
	"net/http"
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
	slog.Debug("Listing quotes")
	quotes, err := h.Store.ListQuotes()
	if err != nil {
		slog.Error("Failed to list quotes", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	views.QuoteList(quotes).Render(r.Context(), w)
}

func (h *QuoteHandler) New(w http.ResponseWriter, r *http.Request) {
	slog.Debug("Rendering new quote form")
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

	views.QuoteForm(quote, customers, products).Render(r.Context(), w)
}

func (h *QuoteHandler) Create(w http.ResponseWriter, r *http.Request) {
	quote, err := h.parseForm(r)
	if err != nil {
		slog.Error("Failed to parse quote form", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	slog.Info("Creating quote", "quote_number", quote.QuoteNumber)

	// Increment quote number if it matches the auto-generated one
	settings, _ := h.Store.GetAppSettings()
	expectedNum := models.FormatDocumentNumber(settings.QuoteNumberSchema, settings.NextQuoteNumber)
	if quote.QuoteNumber == expectedNum {
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
	id, _ := strconv.Atoi(idStr)

	slog.Debug("Editing quote", "id", id)
	quote, err := h.Store.GetQuote(id)
	if err != nil {
		slog.Error("Quote not found for edit", "id", id, "error", err)
		http.Error(w, "Quote not found", http.StatusNotFound)
		return
	}

	customers, _ := h.Store.ListCustomers()
	products, _ := h.Store.ListProducts()

	views.QuoteForm(quote, customers, products).Render(r.Context(), w)
}

func (h *QuoteHandler) Update(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, _ := strconv.Atoi(idStr)

	quote, err := h.parseForm(r)
	if err != nil {
		slog.Error("Failed to parse quote update form", "id", id, "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	quote.ID = id

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
	id, _ := strconv.Atoi(idStr)

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

	invID, err := h.Store.CreateInvoice(invoice)
	if err != nil {
		slog.Error("Failed to create invoice from quote", "quote_id", id, "error", err)
		http.Error(w, "Failed to create invoice: "+err.Error(), http.StatusInternalServerError)
		return
	}

	slog.Info("Invoice created from quote", "quote_id", id, "invoice_id", invID, "invoice_number", invoice.InvoiceNumber)

	// 2. Update Quote Status
	quote.Status = "Umgewandelt"
	h.Store.UpdateQuote(quote)

	// 3. Redirect to new Invoice
	http.Redirect(w, r, fmt.Sprintf("/invoices/%d", invID), http.StatusSeeOther)
}

func (h *QuoteHandler) parseForm(r *http.Request) (*models.Quote, error) {
	if err := r.ParseForm(); err != nil {
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

	return quote, nil
}
