package handlers

import (
	"din-invoice/models"
	"din-invoice/views"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type BackupHandler struct {
	Store  *models.Store
	DBPath string
}

func NewBackupHandler(store *models.Store, dbPath string) *BackupHandler {
	return &BackupHandler{Store: store, DBPath: dbPath}
}

func (h *BackupHandler) List(w http.ResponseWriter, r *http.Request) {
	slog.Debug("Listing backups")
	backups, err := h.Store.ListBackups()
	if err != nil {
		slog.Error("Failed to list backups", "error", err)
		backups = nil
	}
	settings, _ := h.Store.GetAppSettings()
	views.BackupList(backups, settings).Render(r.Context(), w)
}

func (h *BackupHandler) Create(w http.ResponseWriter, r *http.Request) {
	slog.Info("Creating manual backup")
	path, err := h.Store.CreateBackup(h.DBPath)
	if err != nil {
		slog.Error("Failed to create manual backup", "error", err)
		http.Error(w, fmt.Sprintf("Backup fehlgeschlagen: %v", err), http.StatusInternalServerError)
		return
	}
	slog.Info("Backup created successfully", "path", path)
	http.Redirect(w, r, "/backups", http.StatusSeeOther)
}

func (h *BackupHandler) Download(w http.ResponseWriter, r *http.Request) {
	filename := chi.URLParam(r, "filename")
	slog.Info("Downloading backup", "filename", filename)
	path, err := h.Store.GetBackupPath(filename)
	if err != nil {
		slog.Error("Backup not found for download", "filename", filename, "error", err)
		http.Error(w, "Backup nicht gefunden", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeFile(w, r, path)
}

func (h *BackupHandler) Delete(w http.ResponseWriter, r *http.Request) {
	filename := chi.URLParam(r, "filename")
	slog.Info("Deleting backup", "filename", filename)
	if err := h.Store.DeleteBackup(filename); err != nil {
		slog.Error("Failed to delete backup", "filename", filename, "error", err)
		http.Error(w, "Löschen fehlgeschlagen", http.StatusInternalServerError)
		return
	}
	slog.Info("Backup deleted successfully", "filename", filename)
	http.Redirect(w, r, "/backups", http.StatusSeeOther)
}

func (h *BackupHandler) Restore(w http.ResponseWriter, r *http.Request) {
	filename := chi.URLParam(r, "filename")
	slog.Info("Restoring backup", "filename", filename)
	if err := h.Store.RestoreBackup(filename, h.DBPath); err != nil {
		slog.Error("Failed to restore backup", "filename", filename, "error", err)
		http.Error(w, fmt.Sprintf("Wiederherstellung fehlgeschlagen: %v", err), http.StatusInternalServerError)
		return
	}
	slog.Info("Backup restored successfully", "filename", filename)
	http.Redirect(w, r, "/backups?restored=1", http.StatusSeeOther)
}
