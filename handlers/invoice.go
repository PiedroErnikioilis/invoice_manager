package handlers

import (
	"din-invoice/models"
	"din-invoice/services"
	"din-invoice/views"
	"fmt"
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

func (h *InvoiceHandler) List(w http.ResponseWriter, r *http.Request) {
	invoices, err := h.Store.ListInvoices()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	views.InvoiceList(invoices).Render(r.Context(), w)
}

func (h *InvoiceHandler) New(w http.ResponseWriter, r *http.Request) {
	settings, _ := h.Store.GetAppSettings()
	
	// Format invoice number with 4 digits padding
	invNum := fmt.Sprintf("%04d", settings.NextInvoiceNumber)
	
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

	taxRate, _ := strconv.ParseFloat(r.FormValue("tax_rate"), 64)
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
		p, _ := strconv.ParseFloat(prices[i], 64)
		
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
	expectedNum := fmt.Sprintf("%04d", settings.NextInvoiceNumber)
	if invoice.InvoiceNumber == expectedNum || invoice.InvoiceNumber == strconv.Itoa(settings.NextInvoiceNumber) {
		h.Store.IncrementNextInvoiceNumber()
	}

	id, err := h.Store.CreateInvoice(invoice)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

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

	taxRate, _ := strconv.ParseFloat(r.FormValue("tax_rate"), 64)
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
		p, _ := strconv.ParseFloat(prices[i], 64)
		
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

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
		http.Error(w, "Invoice not found", http.StatusNotFound)
		return
	}
	
	settings, err := h.Store.GetAppSettings()
	if err != nil {
		// Log error but continue
	}

	views.InvoiceView(invoice, settings).Render(r.Context(), w)
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
