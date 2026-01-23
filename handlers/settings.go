package handlers

import (
	"din-invoice/models"
	"din-invoice/views"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)

type SettingsHandler struct {
	Store *models.Store
}

func NewSettingsHandler(store *models.Store) *SettingsHandler {
	return &SettingsHandler{Store: store}
}

func (h *SettingsHandler) View(w http.ResponseWriter, r *http.Request) {
	settings, err := h.Store.GetAppSettings()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	views.SettingsForm(settings).Render(r.Context(), w)
}

func (h *SettingsHandler) Save(w http.ResponseWriter, r *http.Request) {
	// ParseMultipartForm to handle file uploads
	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10MB max
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	nextNum, _ := strconv.Atoi(r.FormValue("next_invoice_number"))

	settings := models.AppSettings{
		SenderName:           r.FormValue("sender_name"),
		SenderAddress:        r.FormValue("sender_address"),
		NextInvoiceNumber:    nextNum,
		BankName:             r.FormValue("bank_name"),
		IBAN:                 r.FormValue("iban"),
		BIC:                  r.FormValue("bic"),
		Website:              r.FormValue("website"),
		Email:                r.FormValue("email"),
		PDFOutputPath:        r.FormValue("pdf_output_path"),
		DefaultSmallBusiness: r.FormValue("default_small_business") == "on",
	}

	// Handle Logo Upload
	file, handler, err := r.FormFile("logo")
	if err == nil {
		defer file.Close()

		// Create uploads dir
		uploadDir := "uploads"
		if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
			os.Mkdir(uploadDir, 0755)
		}

		// Generate file path (keep original extension)
		ext := filepath.Ext(handler.Filename)
		filename := "logo" + ext
		filePath := filepath.Join(uploadDir, filename)

		dst, err := os.Create(filePath)
		if err != nil {
			http.Error(w, "Failed to create logo file", http.StatusInternalServerError)
			return
		}
		defer dst.Close()

		if _, err := io.Copy(dst, file); err != nil {
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
