package models

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"
)

// GetSetting retrieves a setting value by key. Returns empty string if not found.
func (s *Store) GetSetting(key string) (string, error) {
	slog.Debug("Executing GetSetting", "key", key)
	var value string
	err := s.DB.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&value)
	if err != nil {
		slog.Debug("Setting not found or error", "key", key, "error", err)
		return "", nil // Return empty if not found, or handle error?
	}
	slog.Debug("Setting retrieved", "key", key, "value", value)
	return value, nil
}

// SetSetting saves or updates a setting.
func (s *Store) SetSetting(key, value string) error {
	slog.Debug("Executing SetSetting", "key", key, "value", value)
	_, err := s.DB.Exec(`INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)`, key, value)
	if err != nil {
		slog.Error("Failed to set setting", "key", key, "error", err)
	} else {
		slog.Debug("Setting saved successfully", "key", key)
	}
	return err
}

type AppSettings struct {
	SenderName              string
	SenderAddress           string
	NextInvoiceNumber       int
	InvoiceNumberSchema     string
	NextQuoteNumber         int
	QuoteNumberSchema       string
	NextCreditNoteNumber    int
	CreditNoteNumberSchema  string
	NextCustomerID          int
	CustomerIDSchema        string
	EuerFilenameSchema      string
	InvoiceFilenameSchema   string
	QuoteFilenameSchema     string
	CreditNoteFilenameSchema string
	InventoryFilenameSchema string
	BankName                string
	IBAN                    string
	BIC                     string
	Website                 string
	Email                   string
	PDFOutputPath           string
	LogoPath                string
	DefaultSmallBusiness    bool
	BackupPath              string
	BackupMaxCount          int
	AutoBackupEnabled       bool
	BackupMinIntervalHours  int
}

func (s AppSettings) BackupMinIntervalHoursStr() string {
	if s.BackupMinIntervalHours == 0 {
		return strconv.Itoa(DefaultBackupMinIntervalHours)
	}
	return strconv.Itoa(s.BackupMinIntervalHours)
}

func (s AppSettings) BackupMaxCountStr() string {
	if s.BackupMaxCount == 0 {
		return strconv.Itoa(DefaultBackupMaxCount)
	}
	return strconv.Itoa(s.BackupMaxCount)
}

func (s *Store) GetAppSettings() (AppSettings, error) {
	slog.Debug("Executing GetAppSettings")
	var settings AppSettings

	val, _ := s.GetSetting("sender_name")
	settings.SenderName = val

	val, _ = s.GetSetting("sender_address")
	settings.SenderAddress = val

	val, _ = s.GetSetting("next_invoice_number")
	if val != "" {
		num, _ := strconv.Atoi(val)
		settings.NextInvoiceNumber = num
	} else {
		settings.NextInvoiceNumber = 1 // Default start
	}

	val, _ = s.GetSetting("invoice_number_schema")
	if val == "" {
		val = "{N:4}"
	}
	settings.InvoiceNumberSchema = val

	val, _ = s.GetSetting("next_quote_number")
	if val != "" {
		num, _ := strconv.Atoi(val)
		settings.NextQuoteNumber = num
	} else {
		settings.NextQuoteNumber = 1
	}

	val, _ = s.GetSetting("quote_number_schema")
	if val == "" {
		val = "AG-{N:4}"
	}
	settings.QuoteNumberSchema = val

	val, _ = s.GetSetting("next_credit_note_number")
	if val != "" {
		num, _ := strconv.Atoi(val)
		settings.NextCreditNoteNumber = num
	} else {
		settings.NextCreditNoteNumber = 1
	}

	val, _ = s.GetSetting("credit_note_number_schema")
	if val == "" {
		val = "GS-{N:4}"
	}
	settings.CreditNoteNumberSchema = val

	val, _ = s.GetSetting("next_customer_id")
	if val != "" {
		num, _ := strconv.Atoi(val)
		settings.NextCustomerID = num
	} else {
		settings.NextCustomerID = 1
	}

	val, _ = s.GetSetting("customer_id_schema")
	if val == "" {
		val = "KD-{N:4}"
	}
	settings.CustomerIDSchema = val

	val, _ = s.GetSetting("euer_filename_schema")
	if val == "" {
		val = "EÜR-{YYYY}"
	}
	settings.EuerFilenameSchema = val

	val, _ = s.GetSetting("invoice_filename_schema")
	if val == "" {
		val = "{ID}"
	}
	settings.InvoiceFilenameSchema = val

	val, _ = s.GetSetting("quote_filename_schema")
	if val == "" {
		val = "Angebot_{ID}"
	}
	settings.QuoteFilenameSchema = val

	val, _ = s.GetSetting("credit_note_filename_schema")
	if val == "" {
		val = "Gutschrift_{ID}"
	}
	settings.CreditNoteFilenameSchema = val

	val, _ = s.GetSetting("inventory_filename_schema")
	if val == "" {
		val = "Inventarliste_{YYYY}-{MM}-{DD}"
	}
	settings.InventoryFilenameSchema = val

	val, _ = s.GetSetting("default_small_business")
	settings.DefaultSmallBusiness = val == "true"

	val, _ = s.GetSetting("bank_name")
	settings.BankName = val

	val, _ = s.GetSetting("iban")
	settings.IBAN = val

	val, _ = s.GetSetting("bic")
	settings.BIC = val

	val, _ = s.GetSetting("website")
	settings.Website = val

	val, _ = s.GetSetting("email")
	settings.Email = val

	val, _ = s.GetSetting("pdf_output_path")
	if val == "" {
		val = "./invoices/" // Default
	}
	settings.PDFOutputPath = val

	val, _ = s.GetSetting("logo_path")
	settings.LogoPath = val

	val, _ = s.GetSetting("backup_path")
	if val == "" {
		val = DefaultBackupPath
	}
	settings.BackupPath = val

	val, _ = s.GetSetting("backup_max_count")
	if val != "" {
		num, _ := strconv.Atoi(val)
		settings.BackupMaxCount = num
	} else {
		settings.BackupMaxCount = DefaultBackupMaxCount
	}

	val, _ = s.GetSetting("auto_backup_enabled")
	settings.AutoBackupEnabled = val != "false" // Default: eingeschaltet

	val, _ = s.GetSetting("backup_min_interval_hours")
	if val != "" {
		num, _ := strconv.Atoi(val)
		settings.BackupMinIntervalHours = num
	} else {
		settings.BackupMinIntervalHours = DefaultBackupMinIntervalHours
	}

	slog.Debug("App settings loaded successfully")
	return settings, nil
}

func (s *Store) SaveAppSettings(settings AppSettings) error {
	slog.Debug("Executing SaveAppSettings")
	if err := s.SetSetting("sender_name", settings.SenderName); err != nil {
		return err
	}
	if err := s.SetSetting("sender_address", settings.SenderAddress); err != nil {
		return err
	}
	if err := s.SetSetting("next_invoice_number", strconv.Itoa(settings.NextInvoiceNumber)); err != nil {
		return err
	}
	if err := s.SetSetting("invoice_number_schema", settings.InvoiceNumberSchema); err != nil {
		return err
	}
	if err := s.SetSetting("next_quote_number", strconv.Itoa(settings.NextQuoteNumber)); err != nil {
		return err
	}
	if err := s.SetSetting("quote_number_schema", settings.QuoteNumberSchema); err != nil {
		return err
	}
	if err := s.SetSetting("next_credit_note_number", strconv.Itoa(settings.NextCreditNoteNumber)); err != nil {
		return err
	}
	if err := s.SetSetting("credit_note_number_schema", settings.CreditNoteNumberSchema); err != nil {
		return err
	}
	if err := s.SetSetting("next_customer_id", strconv.Itoa(settings.NextCustomerID)); err != nil {
		return err
	}
	if err := s.SetSetting("customer_id_schema", settings.CustomerIDSchema); err != nil {
		return err
	}
	if err := s.SetSetting("euer_filename_schema", settings.EuerFilenameSchema); err != nil {
		return err
	}
	if err := s.SetSetting("invoice_filename_schema", settings.InvoiceFilenameSchema); err != nil {
		return err
	}
	if err := s.SetSetting("quote_filename_schema", settings.QuoteFilenameSchema); err != nil {
		return err
	}
	if err := s.SetSetting("credit_note_filename_schema", settings.CreditNoteFilenameSchema); err != nil {
		return err
	}
	if err := s.SetSetting("inventory_filename_schema", settings.InventoryFilenameSchema); err != nil {
		return err
	}
	if err := s.SetSetting("bank_name", settings.BankName); err != nil {
		return err
	}
	if err := s.SetSetting("iban", settings.IBAN); err != nil {
		return err
	}
	if err := s.SetSetting("bic", settings.BIC); err != nil {
		return err
	}
	if err := s.SetSetting("website", settings.Website); err != nil {
		return err
	}
	if err := s.SetSetting("email", settings.Email); err != nil {
		return err
	}
	if err := s.SetSetting("pdf_output_path", settings.PDFOutputPath); err != nil {
		return err
	}
	if err := s.SetSetting("logo_path", settings.LogoPath); err != nil {
		return err
	}
	if err := s.SetSetting("default_small_business", strconv.FormatBool(settings.DefaultSmallBusiness)); err != nil {
		return err
	}
	if err := s.SetSetting("backup_path", settings.BackupPath); err != nil {
		return err
	}
	if err := s.SetSetting("backup_max_count", strconv.Itoa(settings.BackupMaxCount)); err != nil {
		return err
	}
	if err := s.SetSetting("auto_backup_enabled", strconv.FormatBool(settings.AutoBackupEnabled)); err != nil {
		return err
	}
	if err := s.SetSetting("backup_min_interval_hours", strconv.Itoa(settings.BackupMinIntervalHours)); err != nil {
		return err
	}
	slog.Info("App settings saved successfully")
	return nil
}

// IncrementNextInvoiceNumber increments the counter in the DB
func (s *Store) IncrementNextInvoiceNumber() error {
	slog.Info("Incrementing next invoice number")
	settings, err := s.GetAppSettings()
	if err != nil {
		slog.Error("Failed to get settings for invoice increment", "error", err)
		return err
	}
	settings.NextInvoiceNumber++
	slog.Debug("New next invoice number", "value", settings.NextInvoiceNumber)
	return s.SaveAppSettings(settings)
}

// IncrementNextQuoteNumber increments the quote counter in the DB
func (s *Store) IncrementNextQuoteNumber() error {
	slog.Info("Incrementing next quote number")
	settings, err := s.GetAppSettings()
	if err != nil {
		slog.Error("Failed to get settings for quote increment", "error", err)
		return err
	}
	settings.NextQuoteNumber++
	slog.Debug("New next quote number", "value", settings.NextQuoteNumber)
	return s.SaveAppSettings(settings)
}

// IncrementNextCreditNoteNumber increments the credit note counter in the DB
func (s *Store) IncrementNextCreditNoteNumber() error {
	slog.Info("Incrementing next credit note number")
	settings, err := s.GetAppSettings()
	if err != nil {
		slog.Error("Failed to get settings for credit note increment", "error", err)
		return err
	}
	settings.NextCreditNoteNumber++
	slog.Debug("New next credit note number", "value", settings.NextCreditNoteNumber)
	return s.SaveAppSettings(settings)
}

// IncrementNextCustomerID increments the customer counter in the DB
func (s *Store) IncrementNextCustomerID() error {
	slog.Info("Incrementing next customer ID")
	settings, err := s.GetAppSettings()
	if err != nil {
		slog.Error("Failed to get settings for customer ID increment", "error", err)
		return err
	}
	settings.NextCustomerID++
	slog.Debug("New next customer ID", "value", settings.NextCustomerID)
	return s.SaveAppSettings(settings)
}

// FormatFilename formats a filename based on schema, date and document ID.
func FormatFilename(schema string, docID string) string {
	res := FormatDocumentNumber(schema, 0) // Handles {YYYY}, {MM}, {DD}
	return strings.ReplaceAll(res, "{ID}", docID)
}

// FormatDocumentNumber formats a document number based on the schema and counter.
// Supported placeholders:
//
//	{YYYY} - Year 4-digit, {YY} - Year 2-digit
//	{MM} - Month, {DD} - Day
//	{N} - Counter without padding
//	{N:2}..{N:6} - Counter with zero-padding
func FormatDocumentNumber(schema string, number int) string {
	now := time.Now()
	r := strings.NewReplacer(
		"{YYYY}", now.Format("2006"),
		"{YY}", now.Format("06"),
		"{MM}", fmt.Sprintf("%02d", now.Month()),
		"{DD}", fmt.Sprintf("%02d", now.Day()),
	)
	result := r.Replace(schema)

	// Handle {N} and {N:X} placeholders
	if strings.Contains(result, "{N") {
		idx := strings.Index(result, "{N")
		end := strings.Index(result[idx:], "}")
		if end != -1 {
			placeholder := result[idx : idx+end+1]
			var formatted string
			if placeholder == "{N}" {
				formatted = strconv.Itoa(number)
			} else {
				// Parse {N:X} where X is padding width
				var width int
				fmt.Sscanf(placeholder, "{N:%d}", &width)
				if width > 0 {
					formatted = fmt.Sprintf("%0*d", width, number)
				} else {
					formatted = strconv.Itoa(number)
				}
			}
			result = strings.Replace(result, placeholder, formatted, 1)
		}
	}

	return result
}
