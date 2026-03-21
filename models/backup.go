package models

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	DefaultBackupPath             = "./backups"
	DefaultBackupMaxCount         = 10
	DefaultBackupMinIntervalHours = 24
)

type BackupInfo struct {
	Filename  string
	Path      string
	Size      int64
	CreatedAt time.Time
}

// copyFile copies a file from src to dst. On error the destination is removed.
func copyFile(srcPath, dstPath string) (int64, error) {
	slog.Debug("Copying file", "src", srcPath, "dst", dstPath)
	src, err := os.Open(srcPath)
	if err != nil {
		slog.Error("Failed to open source file for copying", "path", srcPath, "error", err)
		return 0, err
	}
	defer src.Close()

	dst, err := os.Create(dstPath)
	if err != nil {
		slog.Error("Failed to create destination file for copying", "path", dstPath, "error", err)
		return 0, err
	}
	defer dst.Close()

	n, err := io.Copy(dst, src)
	if err != nil {
		slog.Error("Failed to copy file contents", "src", srcPath, "dst", dstPath, "error", err)
		os.Remove(dstPath)
		return 0, err
	}
	slog.Debug("File copied successfully", "src", srcPath, "dst", dstPath, "bytes", n)
	return n, nil
}

// validatedBackupPath validates a filename against path traversal and returns the full path.
func (s *Store) validatedBackupPath(filename string) (string, error) {
	slog.Debug("Validating backup filename", "filename", filename)
	cleaned := filepath.Base(filename)
	if cleaned != filename || strings.Contains(filename, "..") {
		slog.Error("Invalid backup filename detected", "filename", filename)
		return "", fmt.Errorf("ungültiger Dateiname")
	}
	settings, _ := s.GetAppSettings()
	return filepath.Join(settings.BackupPath, cleaned), nil
}

func (s *Store) CreateBackup(dbPath string) (*BackupInfo, error) {
	slog.Debug("Executing CreateBackup", "db_path", dbPath)
	settings, _ := s.GetAppSettings()
	slog.Info("Creating database backup", "db_path", dbPath, "backup_dir", settings.BackupPath)
	if err := os.MkdirAll(settings.BackupPath, 0755); err != nil {
		slog.Error("Failed to create backup directory", "path", settings.BackupPath, "error", err)
		return nil, fmt.Errorf("backup-Verzeichnis erstellen: %w", err)
	}

	// Ensure WAL is flushed before copying
	slog.Debug("Flushing WAL before backup")
	s.DB.Exec("PRAGMA wal_checkpoint(TRUNCATE)")

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("backup_%s.db", timestamp)
	destPath := filepath.Join(settings.BackupPath, filename)

	n, err := copyFile(dbPath, destPath)
	if err != nil {
		slog.Error("Failed to copy database file for backup", "src", dbPath, "dst", destPath, "error", err)
		return nil, fmt.Errorf("Datenbank kopieren: %w", err)
	}

	// Rotate old backups
	slog.Debug("Rotating backups after creation")
	s.rotateBackups(settings)

	slog.Info("Backup created successfully", "filename", filename, "size", n)
	return &BackupInfo{
		Filename:  filename,
		Path:      destPath,
		Size:      n,
		CreatedAt: time.Now(),
	}, nil
}

func (s *Store) ListBackups() ([]BackupInfo, error) {
	slog.Debug("Executing ListBackups")
	settings, _ := s.GetAppSettings()
	slog.Debug("Listing backups from directory", "path", settings.BackupPath)
	entries, err := os.ReadDir(settings.BackupPath)
	if err != nil {
		if os.IsNotExist(err) {
			slog.Debug("Backup directory does not exist", "path", settings.BackupPath)
			return nil, nil
		}
		slog.Error("Failed to read backup directory", "path", settings.BackupPath, "error", err)
		return nil, err
	}

	var backups []BackupInfo
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".db") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			slog.Error("Failed to get file info for backup entry", "name", e.Name(), "error", err)
			continue
		}
		backups = append(backups, BackupInfo{
			Filename:  e.Name(),
			Path:      filepath.Join(settings.BackupPath, e.Name()),
			Size:      info.Size(),
			CreatedAt: info.ModTime(),
		})
	}

	sort.Slice(backups, func(i, j int) bool {
		return backups[i].CreatedAt.After(backups[j].CreatedAt)
	})

	slog.Debug("Backups listed successfully", "count", len(backups))
	return backups, nil
}

func (s *Store) DeleteBackup(filename string) error {
	slog.Debug("Executing DeleteBackup", "filename", filename)
	slog.Info("Deleting backup file", "filename", filename)
	path, err := s.validatedBackupPath(filename)
	if err != nil {
		slog.Error("Invalid backup filename for deletion", "filename", filename, "error", err)
		return err
	}
	if err := os.Remove(path); err != nil {
		slog.Error("Failed to remove backup file", "path", path, "error", err)
		return err
	}
	slog.Info("Backup file deleted", "path", path)
	return nil
}

func (s *Store) GetBackupPath(filename string) (string, error) {
	slog.Debug("Executing GetBackupPath", "filename", filename)
	path, err := s.validatedBackupPath(filename)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(path); err != nil {
		slog.Error("Backup file not found", "path", path, "error", err)
		return "", fmt.Errorf("Backup nicht gefunden")
	}
	return path, nil
}

func (s *Store) RestoreBackup(filename, dbPath string) error {
	slog.Debug("Executing RestoreBackup", "filename", filename, "db_path", dbPath)
	slog.Info("Restoring database from backup", "filename", filename, "target", dbPath)
	backupPath, err := s.GetBackupPath(filename)
	if err != nil {
		slog.Error("Failed to get backup path for restore", "filename", filename, "error", err)
		return err
	}

	settings, _ := s.GetAppSettings()
	if err := os.MkdirAll(settings.BackupPath, 0755); err != nil {
		slog.Error("Failed to create safety backup directory", "path", settings.BackupPath, "error", err)
		return fmt.Errorf("Sicherheitsbackup-Verzeichnis: %w", err)
	}

	// Safety backup before restoring
	safetyName := fmt.Sprintf("vor_wiederherstellung_%s.db", time.Now().Format("2006-01-02_15-04-05"))
	safetyPath := filepath.Join(settings.BackupPath, safetyName)

	slog.Info("Creating safety backup before restore", "path", safetyPath)
	if _, err := copyFile(dbPath, safetyPath); err != nil {
		slog.Error("Failed to create safety backup", "path", safetyPath, "error", err)
		return fmt.Errorf("Sicherheitsbackup erstellen: %w", err)
	}

	// Restore: copy backup over current DB
	slog.Debug("Overwriting database with backup file")
	if _, err := copyFile(backupPath, dbPath); err != nil {
		slog.Error("Failed to restore backup over database", "src", backupPath, "dst", dbPath, "error", err)
		return fmt.Errorf("Backup wiederherstellen: %w", err)
	}

	slog.Info("Database restored successfully from backup", "filename", filename)
	return nil
}

func (s *Store) rotateBackups(settings AppSettings) {
	maxCount := settings.BackupMaxCount
	if maxCount < 1 {
		maxCount = DefaultBackupMaxCount
	}
	slog.Debug("Executing rotateBackups", "max_count", maxCount)

	backups, err := s.ListBackups()
	if err != nil {
		slog.Error("Failed to list backups for rotation", "error", err)
		return
	}

	if len(backups) <= maxCount {
		slog.Debug("No backup rotation needed", "current_count", len(backups), "max_count", maxCount)
		return
	}

	slog.Debug("Rotating old backups", "count_to_delete", len(backups)-maxCount)
	for _, b := range backups[maxCount:] {
		slog.Info("Deleting old backup during rotation", "path", b.Path)
		if err := os.Remove(b.Path); err != nil {
			slog.Error("Failed to delete old backup during rotation", "path", b.Path, "error", err)
		}
	}
	slog.Debug("Backup rotation finished")
}

func FormatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
	)
	switch {
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
