package models

import (
	"strconv"
)

// GetSetting retrieves a setting value by key. Returns empty string if not found.
func (s *Store) GetSetting(key string) (string, error) {
	var value string
	err := s.DB.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&value)
	if err != nil {
		return "", nil // Return empty if not found, or handle error?
	}
	return value, nil
}

// SetSetting saves or updates a setting.
func (s *Store) SetSetting(key, value string) error {
	_, err := s.DB.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)", key, value)
	return err
}

type AppSettings struct {
	SenderName           string
	SenderAddress        string
	NextInvoiceNumber    int
	BankName             string
	IBAN                 string
	BIC                  string
	Website              string
	Email                string
	PDFOutputPath        string
	LogoPath             string
	DefaultSmallBusiness bool
	BackupPath             string
	BackupMaxCount         int
	AutoBackupEnabled      bool
	BackupMinIntervalHours int
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

	return settings, nil
}

func (s *Store) SaveAppSettings(settings AppSettings) error {
	if err := s.SetSetting("sender_name", settings.SenderName); err != nil {
		return err
	}
	if err := s.SetSetting("sender_address", settings.SenderAddress); err != nil {
		return err
	}
	if err := s.SetSetting("next_invoice_number", strconv.Itoa(settings.NextInvoiceNumber)); err != nil {
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
	return nil
}

// IncrementNextInvoiceNumber increments the counter in the DB
func (s *Store) IncrementNextInvoiceNumber() error {
	settings, err := s.GetAppSettings()
	if err != nil {
		return err
	}
	settings.NextInvoiceNumber++
	return s.SaveAppSettings(settings)
}
