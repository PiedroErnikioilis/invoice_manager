package handlers

import (
	"din-invoice/models"
	"din-invoice/views"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)

type SettingsHandler struct {
	Store *models.Store
}

func NewSettingsHandler(s *models.Store) *SettingsHandler {
	return &SettingsHandler{Store: s}
}

func (h *SettingsHandler) View(w http.ResponseWriter, r *http.Request) {
	slog.Debug("Processing View settings request", "method", r.Method)
	settings, err := h.Store.GetAppSettings()
	if err != nil {
		slog.Error("Failed to load settings", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	slog.Debug("Settings loaded successfully")
	views.Settings(settings).Render(r.Context(), w)
}

func (h *SettingsHandler) Save(w http.ResponseWriter, r *http.Request) {
	slog.Debug("Processing Save settings request", "method", r.Method)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		slog.Error("Failed to parse form", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	nextNum, _ := strconv.Atoi(r.FormValue("next_invoice_number"))
	nextQuoteNum, _ := strconv.Atoi(r.FormValue("next_quote_number"))
	nextCreditNoteNum, _ := strconv.Atoi(r.FormValue("next_credit_note_number"))
	nextCustomerID, _ := strconv.Atoi(r.FormValue("next_customer_id"))

	slog.Debug("Settings form parsed", 
		"next_inv", nextNum, 
		"next_quote", nextQuoteNum, 
		"next_customer", nextCustomerID)

	backupMaxCount, _ := strconv.Atoi(r.FormValue("backup_max_count"))
	if backupMaxCount < 1 {
		backupMaxCount = models.DefaultBackupMaxCount
	}
	backupMinInterval, _ := strconv.Atoi(r.FormValue("backup_min_interval"))
	if backupMinInterval < 1 {
		backupMinInterval = 24
	}

	backupPath := r.FormValue("backup_path")
	if backupPath == "" {
		backupPath = "./backups"
	}

	invoiceSchema := r.FormValue("invoice_number_schema")
	if invoiceSchema == "" {
		invoiceSchema = "{N:4}"
	}
	quoteSchema := r.FormValue("quote_number_schema")
	if quoteSchema == "" {
		quoteSchema = "AG-{N:4}"
	}
	creditNoteSchema := r.FormValue("credit_note_number_schema")
	if creditNoteSchema == "" {
		creditNoteSchema = "GS-{N:4}"
	}
	customerIDSchema := r.FormValue("customer_id_schema")
	if customerIDSchema == "" {
		customerIDSchema = "KD-{N:4}"
	}
	euerFilenameSchema := r.FormValue("euer_filename_schema")
	if euerFilenameSchema == "" {
		euerFilenameSchema = "EÜR-{YYYY}"
	}
	invoiceFilenameSchema := r.FormValue("invoice_filename_schema")
	if invoiceFilenameSchema == "" {
		invoiceFilenameSchema = "{ID}"
	}
	quoteFilenameSchema := r.FormValue("quote_filename_schema")
	if quoteFilenameSchema == "" {
		quoteFilenameSchema = "Angebot_{ID}"
	}
	creditNoteFilenameSchema := r.FormValue("credit_note_filename_schema")
	if creditNoteFilenameSchema == "" {
		creditNoteFilenameSchema = "Gutschrift_{ID}"
	}
	inventoryFilenameSchema := r.FormValue("inventory_filename_schema")
	if inventoryFilenameSchema == "" {
		inventoryFilenameSchema = "Inventarliste_{YYYY}-{MM}-{DD}"
	}

	settings := models.AppSettings{
		SenderName:               r.FormValue("sender_name"),
		SenderAddress:            r.FormValue("sender_address"),
		NextInvoiceNumber:       nextNum,
		InvoiceNumberSchema:     invoiceSchema,
		NextQuoteNumber:         nextQuoteNum,
		QuoteNumberSchema:       quoteSchema,
		NextCreditNoteNumber:    nextCreditNoteNum,
		CreditNoteNumberSchema:  creditNoteSchema,
		NextCustomerID:          nextCustomerID,
		CustomerIDSchema:        customerIDSchema,
		EuerFilenameSchema:      euerFilenameSchema,
		InvoiceFilenameSchema:   invoiceFilenameSchema,
		QuoteFilenameSchema:     quoteFilenameSchema,
		CreditNoteFilenameSchema: creditNoteFilenameSchema,
		InventoryFilenameSchema: inventoryFilenameSchema,
		BankName:                r.FormValue("bank_name"),
		IBAN:                    r.FormValue("iban"),
		BIC:                     r.FormValue("bic"),
		Website:                 r.FormValue("website"),
		Email:                   r.FormValue("email"),
		PDFOutputPath:           r.FormValue("pdf_output_path"),
		LogoPath:                r.FormValue("logo_path"),
		DefaultSmallBusiness:    r.FormValue("default_small_business") == "on",
		BackupPath:              backupPath,
		BackupMaxCount:          backupMaxCount,
		AutoBackupEnabled:       r.FormValue("auto_backup_enabled") == "on",
		BackupMinIntervalHours:  backupMinInterval,
	}

	// Handle Logo Upload
	file, handler, err := r.FormFile("logo")
	if err == nil {
		defer file.Close()
		slog.Debug("Uploading new logo", "filename", handler.Filename)

		// Create uploads dir
		uploadDir := "uploads"
		if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
			os.MkdirAll(uploadDir, 0755)
		}

		// Generate file path (keep original extension)
		ext := filepath.Ext(handler.Filename)
		filename := "logo" + ext
		filePath := filepath.Join(uploadDir, filename)

		dst, err := os.Create(filePath)
		if err != nil {
			slog.Error("Failed to create logo file", "path", filePath, "error", err)
			http.Error(w, "Failed to create logo file", http.StatusInternalServerError)
			return
		}
		defer dst.Close()

		if _, err := io.Copy(dst, file); err != nil {
			slog.Error("Failed to save logo file", "path", filePath, "error", err)
			http.Error(w, "Failed to save logo file", http.StatusInternalServerError)
			return
		}

		settings.LogoPath = filePath
	} else {
		// Keep existing logo if not uploaded
		existing, _ := h.Store.GetAppSettings()
		settings.LogoPath = existing.LogoPath
	}

	if err := h.Store.SaveAppSettings(settings); err != nil {
		slog.Error("Failed to save settings", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	slog.Info("Settings saved successfully")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
