package handlers

import (
	"din-invoice/models"
	"din-invoice/services"
	"din-invoice/views"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

type CreditNoteHandler struct {
	Store *models.Store
}

func NewCreditNoteHandler(store *models.Store) *CreditNoteHandler {
	return &CreditNoteHandler{Store: store}
}

func (h *CreditNoteHandler) List(w http.ResponseWriter, r *http.Request) {
	slog.Debug("Processing List credit notes request", "method", r.Method)
	notes, err := h.Store.ListCreditNotes()
	if err != nil {
		slog.Error("Failed to list credit notes", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	slog.Debug("Successfully listed credit notes", "count", len(notes))
	views.CreditNoteList(notes).Render(r.Context(), w)
}

func (h *CreditNoteHandler) NewFromInvoice(w http.ResponseWriter, r *http.Request) {
	invoiceIDStr := r.URL.Query().Get("invoice_id")
	if invoiceIDStr == "" {
		slog.Error("Invoice ID missing for new credit note", "method", r.Method)
		http.Error(w, "Invoice ID required", http.StatusBadRequest)
		return
	}
	invID, _ := strconv.Atoi(invoiceIDStr)
	slog.Debug("Processing NewFromInvoice credit note request", "invoice_id", invID, "method", r.Method)

	slog.Debug("Fetching invoice for credit note creation", "invoice_id", invID)
	invoice, err := h.Store.GetInvoice(invID)
	if err != nil {
		slog.Error("Failed to load invoice for credit note", "invoice_id", invID, "error", err)
		http.Error(w, "Invoice not found", http.StatusNotFound)
		return
	}

	settings, _ := h.Store.GetAppSettings()
	creditNoteNum := models.FormatDocumentNumber(settings.CreditNoteNumberSchema, settings.NextCreditNoteNumber)

	note := &models.CreditNote{
		CreditNoteNumber: creditNoteNum,
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

	slog.Debug("Successfully prepared credit note form from invoice", "invoice_id", invID, "credit_note_number", creditNoteNum)
	views.CreditNoteForm(note, customers, products).Render(r.Context(), w)
}

func (h *CreditNoteHandler) Create(w http.ResponseWriter, r *http.Request) {
	slog.Debug("Processing Create credit note request", "method", r.Method)
	note, err := h.parseForm(r)
	if err != nil {
		slog.Error("Failed to parse credit note form", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	slog.Debug("Successfully parsed credit note form", "credit_note_number", note.CreditNoteNumber)

	slog.Info("Creating credit note", "credit_note_number", note.CreditNoteNumber)

	// Increment credit note number if it matches the auto-generated one
	settings, _ := h.Store.GetAppSettings()
	expectedNum := models.FormatDocumentNumber(settings.CreditNoteNumberSchema, settings.NextCreditNoteNumber)
	if note.CreditNoteNumber == expectedNum {
		slog.Debug("Incrementing next credit note number", "credit_note_number", note.CreditNoteNumber)
		h.Store.IncrementNextCreditNoteNumber()
	}

	id, err := h.Store.CreateCreditNote(note)
	if err != nil {
		slog.Error("Failed to create credit note", "credit_note_number", note.CreditNoteNumber, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	slog.Info("Credit note created successfully", "id", id, "credit_note_number", note.CreditNoteNumber)
	http.Redirect(w, r, "/credit-notes/"+strconv.Itoa(id), http.StatusSeeOther)
}

func (h *CreditNoteHandler) View(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		slog.Error("Failed to parse credit note ID for view", "id", idStr, "error", err)
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	slog.Debug("Processing View credit note request", "id", id, "method", r.Method)

	slog.Debug("Fetching credit note for view", "id", id)
	note, err := h.Store.GetCreditNote(id)
	if err != nil {
		slog.Error("Credit note not found", "id", id, "error", err)
		http.Error(w, "Credit note not found", http.StatusNotFound)
		return
	}

	settings, _ := h.Store.GetAppSettings()
	slog.Debug("Successfully fetched credit note and settings for view", "id", id)
	views.CreditNoteView(note, settings).Render(r.Context(), w)
}

func (h *CreditNoteHandler) DownloadPDF(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		slog.Error("Failed to parse credit note ID for PDF download", "id", idStr, "error", err)
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	slog.Debug("Processing DownloadPDF credit note request", "id", id, "method", r.Method)

	force := r.URL.Query().Get("force") == "1"

	slog.Debug("Fetching credit note for PDF", "id", id)
	note, err := h.Store.GetCreditNote(id)
	if err != nil {
		slog.Error("Credit note not found for PDF", "id", id, "error", err)
		http.Error(w, "Credit note not found", http.StatusNotFound)
		return
	}

	settings, _ := h.Store.GetAppSettings()
	path := services.GetCreditNotePDFPath(note, &settings)

	// Smart Check: If file exists and state is final, don't regenerate unless forced
	isFinal := note.Status == "Abgeschlossen"
	if !force && isFinal {
		if _, err := os.Stat(path); err == nil {
			slog.Debug("Serving existing credit note PDF", "path", path, "status", note.Status)
			w.Header().Set("Content-Type", "application/pdf")
			w.Header().Set("Content-Disposition", "inline; filename="+filepath.Base(path))
			http.ServeFile(w, r, path)
			return
		}
	}

	slog.Info("Generating fresh credit note PDF", "id", id, "force", force, "is_final", isFinal)
	path, err = services.GenerateCreditNotePDFHTML(note, &settings)
	if err != nil {
		slog.Error("Failed to generate credit note PDF", "id", id, "error", err)
		http.Error(w, "Failed to generate PDF: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "inline; filename="+filepath.Base(path))
	slog.Debug("Serving freshly generated credit note PDF", "path", path)
	http.ServeFile(w, r, path)
}

func (h *CreditNoteHandler) parseForm(r *http.Request) (*models.CreditNote, error) {
	slog.Debug("Parsing credit note form", "method", r.Method)
	if err := r.ParseForm(); err != nil {
		slog.Error("Failed to parse credit note form", "error", err)
		return nil, err
	}

	taxRate := parseDecimal(r.FormValue("tax_rate"))
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

	slog.Debug("Parsing credit note items", "count", len(descriptions))
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

		note.Items = append(note.Items, models.CreditNoteItem{
			Description:  descriptions[i],
			Quantity:     qty,
			PricePerUnit: price,
			ProductID:    pID,
		})
	}

	slog.Debug("Successfully parsed credit note form", "credit_note_number", note.CreditNoteNumber, "items_count", len(note.Items))
	return note, nil
}
