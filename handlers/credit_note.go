package handlers

import (
	"din-invoice/models"
	"din-invoice/views"
	"net/http"
	"strconv"
	"time"
)

type CreditNoteHandler struct {
	Store *models.Store
}

func NewCreditNoteHandler(store *models.Store) *CreditNoteHandler {
	return &CreditNoteHandler{Store: store}
}

func (h *CreditNoteHandler) List(w http.ResponseWriter, r *http.Request) {
	notes, err := h.Store.ListCreditNotes()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	views.CreditNoteList(notes).Render(r.Context(), w)
}

func (h *CreditNoteHandler) NewFromInvoice(w http.ResponseWriter, r *http.Request) {
	invoiceIDStr := r.URL.Query().Get("invoice_id")
	if invoiceIDStr == "" {
		http.Error(w, "Invoice ID required", http.StatusBadRequest)
		return
	}
	invID, _ := strconv.Atoi(invoiceIDStr)

	invoice, err := h.Store.GetInvoice(invID)
	if err != nil {
		http.Error(w, "Invoice not found", http.StatusNotFound)
		return
	}

	note := &models.CreditNote{
		CreditNoteNumber: "GS-" + invoice.InvoiceNumber,
		Date:             time.Now().Format("2006-01-02"),
		SenderName:       invoice.SenderName,
		SenderAddress:    invoice.SenderAddress,
		RecipientName:    invoice.RecipientName,
		RecipientAddress: invoice.RecipientAddress,
		TaxRate:          invoice.TaxRate,
		IsSmallBusiness:  invoice.IsSmallBusiness,
		CustomerID:       invoice.CustomerID,
		InvoiceID:        &invID,
	}

	for _, item := range invoice.Items {
		note.Items = append(note.Items, models.CreditNoteItem{
			Description:  item.Description,
			Quantity:     item.Quantity,
			PricePerUnit: item.PricePerUnit,
			ProductID:    item.ProductID,
		})
	}

	customers, _ := h.Store.ListCustomers()
	products, _ := h.Store.ListProducts()

	views.CreditNoteForm(note, customers, products).Render(r.Context(), w)
}

func (h *CreditNoteHandler) Create(w http.ResponseWriter, r *http.Request) {
	note, err := h.parseForm(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, err = h.Store.CreateCreditNote(note)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/credit-notes", http.StatusSeeOther)
}

func (h *CreditNoteHandler) parseForm(r *http.Request) (*models.CreditNote, error) {
	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	taxRate, _ := strconv.ParseFloat(r.FormValue("tax_rate"), 64)
	customerIDStr := r.FormValue("customer_id")
	var customerID *int
	if customerIDStr != "" {
		id, _ := strconv.Atoi(customerIDStr)
		customerID = &id
	}

	invoiceIDStr := r.FormValue("invoice_id")
	var invoiceID *int
	if invoiceIDStr != "" {
		id, _ := strconv.Atoi(invoiceIDStr)
		invoiceID = &id
	}

	note := &models.CreditNote{
		CreditNoteNumber: r.FormValue("credit_note_number"),
		Date:             r.FormValue("date"),
		SenderName:       r.FormValue("sender_name"),
		SenderAddress:    r.FormValue("sender_address"),
		RecipientName:    r.FormValue("recipient_name"),
		RecipientAddress: r.FormValue("recipient_address"),
		TaxRate:          taxRate,
		Status:           r.FormValue("status"),
		IsSmallBusiness:  r.FormValue("is_small_business") == "true",
		CustomerID:       customerID,
		InvoiceID:        invoiceID,
	}

	descriptions := r.Form["item_description[]"]
	quantities := r.Form["item_quantity[]"]
	prices := r.Form["item_price[]"]
	productIDs := r.Form["item_product_id[]"]

	for i := range descriptions {
		if descriptions[i] == "" {
			continue
		}
		qty, _ := strconv.Atoi(quantities[i])
		price, _ := strconv.ParseFloat(prices[i], 64)
		var pID *int
		if i < len(productIDs) && productIDs[i] != "" {
			id, _ := strconv.Atoi(productIDs[i])
			pID = &id
		}

		note.Items = append(note.Items, models.CreditNoteItem{
			Description:  descriptions[i],
			Quantity:     qty,
			PricePerUnit: price,
			ProductID:    pID,
		})
	}

	return note, nil
}
