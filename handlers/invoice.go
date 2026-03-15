package handlers

import (
	"din-invoice/models"
	"din-invoice/services"
	"din-invoice/views"
	"log/slog"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

type InvoiceHandler struct {
	Store *models.Store
}

func NewInvoiceHandler(store *models.Store) *InvoiceHandler {
	return &InvoiceHandler{Store: store}
}

// syncCustomerFromInvoice updates the customer record's name and address
// from the invoice recipient data. Errors are logged but not fatal since
// the invoice has already been persisted successfully.
func (h *InvoiceHandler) syncCustomerFromInvoice(inv *models.Invoice) {
	if inv.CustomerID == nil {
		return
	}
	if err := h.Store.UpdateCustomer(models.Customer{
		ID:      *inv.CustomerID,
		Name:    inv.RecipientName,
		Address: inv.RecipientAddress,
	}); err != nil {
		slog.Error("Failed to sync customer from invoice", "customer_id", *inv.CustomerID, "invoice_number", inv.InvoiceNumber, "error", err)
	}
}

func (h *InvoiceHandler) List(w http.ResponseWriter, r *http.Request) {
	filter := models.InvoiceFilter{
		Search: r.URL.Query().Get("q"),
		Status: r.URL.Query().Get("status"),
		Sort:   r.URL.Query().Get("sort"),
		Order:  r.URL.Query().Get("order"),
	}

	invoices, err := h.Store.ListInvoices(filter)
	if err != nil {
		slog.Error("Failed to list invoices", "filter", filter, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	views.InvoiceList(invoices, filter).Render(r.Context(), w)
}

func (h *InvoiceHandler) New(w http.ResponseWriter, r *http.Request) {
	settings, _ := h.Store.GetAppSettings()

	// Format invoice number using the configured schema
	invNum := models.FormatDocumentNumber(settings.InvoiceNumberSchema, settings.NextInvoiceNumber)

	invoice := &models.Invoice{
		SenderName:      settings.SenderName,
		SenderAddress:   settings.SenderAddress,
		InvoiceNumber:   invNum,
		Status:          "Entwurf",
		Date:            time.Now().Format("2006-01-02"),
		TaxRate:         19.0,
		IsSmallBusiness: settings.DefaultSmallBusiness,
	}

	// Pre-fill customer if provided
	customerIDStr := r.URL.Query().Get("customer_id")
	if customerIDStr != "" {
		cid, err := strconv.Atoi(customerIDStr)
		if err == nil {
			customer, err := h.Store.GetCustomer(cid)
			if err == nil {
				invoice.CustomerID = &customer.ID
				invoice.RecipientName = customer.Name
				invoice.RecipientAddress = customer.Address
			}
		}
	}

	products, _ := h.Store.ListProducts()
	customers, _ := h.Store.ListCustomers()
	views.InvoiceForm(invoice, products, customers).Render(r.Context(), w)
}

func (h *InvoiceHandler) Create(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	invoice := &models.Invoice{
		InvoiceNumber:    r.FormValue("invoice_number"),
		Date:             r.FormValue("date"),
		SenderName:       r.FormValue("sender_name"),
		SenderAddress:    r.FormValue("sender_address"),
		RecipientName:    r.FormValue("recipient_name"),
		RecipientAddress: r.FormValue("recipient_address"),
		Status:           r.FormValue("status"),
		IsSmallBusiness:  r.FormValue("is_small_business") == "on",
		Items:            []models.InvoiceItem{},
	}

	if cidStr := r.FormValue("customer_id"); cidStr != "" {
		if cid, err := strconv.Atoi(cidStr); err == nil && cid != 0 {
			invoice.CustomerID = &cid
		}
	}

	taxRate := parseDecimal(r.FormValue("tax_rate"))
	invoice.TaxRate = taxRate

	descriptions := r.Form["description[]"]
	quantities := r.Form["quantity[]"]
	prices := r.Form["price[]"]
	productIDs := r.Form["product_id[]"]

	for i := range descriptions {
		if descriptions[i] == "" {
			continue
		}
		q, _ := strconv.Atoi(quantities[i])
		p := parseDecimal(prices[i])

		var pid *int
		if i < len(productIDs) && productIDs[i] != "" {
			id, err := strconv.Atoi(productIDs[i])
			if err == nil && id != 0 {
				pid = &id
			}
		}

		invoice.Items = append(invoice.Items, models.InvoiceItem{
			Description:  descriptions[i],
			Quantity:     q,
			PricePerUnit: p,
			ProductID:    pid,
		})
	}

	// Increment invoice number if it matches the auto-generated one
	settings, _ := h.Store.GetAppSettings()
	expectedNum := models.FormatDocumentNumber(settings.InvoiceNumberSchema, settings.NextInvoiceNumber)
	if invoice.InvoiceNumber == expectedNum {
		h.Store.IncrementNextInvoiceNumber()
	}

	id, err := h.Store.CreateInvoice(invoice)
	if err != nil {
		slog.Error("Failed to create invoice", "invoice_number", invoice.InvoiceNumber, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	slog.Info("Invoice created", "id", id, "invoice_number", invoice.InvoiceNumber)
	// Sync customer name/address from invoice recipient data
	h.syncCustomerFromInvoice(invoice)

	http.Redirect(w, r, "/invoices/"+strconv.Itoa(id), http.StatusSeeOther)
}

func (h *InvoiceHandler) Edit(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	invoice, err := h.Store.GetInvoice(id)
	if err != nil {
		http.Error(w, "Invoice not found", http.StatusNotFound)
		return
	}

	products, _ := h.Store.ListProducts()
	customers, _ := h.Store.ListCustomers()
	views.InvoiceForm(invoice, products, customers).Render(r.Context(), w)
}

func (h *InvoiceHandler) Update(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	invoice := &models.Invoice{
		ID:               id,
		InvoiceNumber:    r.FormValue("invoice_number"),
		Date:             r.FormValue("date"),
		SenderName:       r.FormValue("sender_name"),
		SenderAddress:    r.FormValue("sender_address"),
		RecipientName:    r.FormValue("recipient_name"),
		RecipientAddress: r.FormValue("recipient_address"),
		Status:           r.FormValue("status"),
		IsSmallBusiness:  r.FormValue("is_small_business") == "on",
		Items:            []models.InvoiceItem{},
	}

	if cidStr := r.FormValue("customer_id"); cidStr != "" {
		if cid, err := strconv.Atoi(cidStr); err == nil && cid != 0 {
			invoice.CustomerID = &cid
		}
	}

	taxRate := parseDecimal(r.FormValue("tax_rate"))
	invoice.TaxRate = taxRate

	descriptions := r.Form["description[]"]
	quantities := r.Form["quantity[]"]
	prices := r.Form["price[]"]
	productIDs := r.Form["product_id[]"]

	for i := range descriptions {
		if descriptions[i] == "" {
			continue
		}
		q, _ := strconv.Atoi(quantities[i])
		p := parseDecimal(prices[i])

		var pid *int
		if i < len(productIDs) && productIDs[i] != "" {
			id, err := strconv.Atoi(productIDs[i])
			if err == nil && id != 0 {
				pid = &id
			}
		}

		invoice.Items = append(invoice.Items, models.InvoiceItem{
			Description:  descriptions[i],
			Quantity:     q,
			PricePerUnit: p,
			ProductID:    pid,
		})
	}

	err = h.Store.UpdateInvoice(invoice)
	if err != nil {
		slog.Error("Failed to update invoice", "id", invoice.ID, "invoice_number", invoice.InvoiceNumber, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	slog.Info("Invoice updated", "id", invoice.ID, "invoice_number", invoice.InvoiceNumber)
	// Sync customer name/address from invoice recipient data
	h.syncCustomerFromInvoice(invoice)

	http.Redirect(w, r, "/invoices/"+strconv.Itoa(id), http.StatusSeeOther)
}

func (h *InvoiceHandler) View(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	invoice, err := h.Store.GetInvoice(id)
	if err != nil {
		slog.Error("Invoice not found", "id", id, "error", err)
		http.Error(w, "Invoice not found", http.StatusNotFound)
		return
	}

	settings, err := h.Store.GetAppSettings()
	if err != nil {
		slog.Error("Failed to load settings for view", "error", err)
		// Log error but continue
	}

	views.InvoiceView(invoice, settings).Render(r.Context(), w)
}

func (h *InvoiceHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	err = h.Store.CancelInvoice(id)
	if err != nil {
		slog.Error("Failed to cancel invoice", "id", id, "error", err)
		http.Error(w, "Fehler beim Stornieren", http.StatusInternalServerError)
		return
	}

	slog.Info("Invoice cancelled", "id", id)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *InvoiceHandler) DownloadPDF(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	invoice, err := h.Store.GetInvoice(id)
	if err != nil {
		http.Error(w, "Invoice not found", http.StatusNotFound)
		return
	}

	settings, err := h.Store.GetAppSettings()
	if err != nil {
		http.Error(w, "Could not load settings", http.StatusInternalServerError)
		return
	}

	path, err := services.GenerateInvoicePDFHTML(invoice, &settings)
	if err != nil {
		http.Error(w, "Failed to generate PDF: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "inline; filename="+filepath.Base(path))
	http.ServeFile(w, r, path)
}
