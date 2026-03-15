package models

import (
	"fmt"
	"io"
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
	src, err := os.Open(srcPath)
	if err != nil {
		return 0, err
	}
	defer src.Close()

	dst, err := os.Create(dstPath)
	if err != nil {
		return 0, err
	}
	defer dst.Close()

	n, err := io.Copy(dst, src)
	if err != nil {
		os.Remove(dstPath)
		return 0, err
	}
	return n, nil
}

// validatedBackupPath validates a filename against path traversal and returns the full path.
func (s *Store) validatedBackupPath(filename string) (string, error) {
	cleaned := filepath.Base(filename)
	if cleaned != filename || strings.Contains(filename, "..") {
		return "", fmt.Errorf("ungültiger Dateiname")
	}
	settings, _ := s.GetAppSettings()
	return filepath.Join(settings.BackupPath, cleaned), nil
}

func (s *Store) CreateBackup(dbPath string) (*BackupInfo, error) {
	settings, _ := s.GetAppSettings()
	if err := os.MkdirAll(settings.BackupPath, 0755); err != nil {
		return nil, fmt.Errorf("backup-Verzeichnis erstellen: %w", err)
	}

	// Ensure WAL is flushed before copying
	s.DB.Exec("PRAGMA wal_checkpoint(TRUNCATE)")

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("backup_%s.db", timestamp)
	destPath := filepath.Join(settings.BackupPath, filename)

	n, err := copyFile(dbPath, destPath)
	if err != nil {
		return nil, fmt.Errorf("Datenbank kopieren: %w", err)
	}

	// Rotate old backups
	s.rotateBackups(settings)

	return &BackupInfo{
		Filename:  filename,
		Path:      destPath,
		Size:      n,
		CreatedAt: time.Now(),
	}, nil
}

func (s *Store) ListBackups() ([]BackupInfo, error) {
	settings, _ := s.GetAppSettings()
	entries, err := os.ReadDir(settings.BackupPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var backups []BackupInfo
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".db") {
			continue
		}
		info, err := e.Info()
		if err != nil {
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

	return backups, nil
}

func (s *Store) DeleteBackup(filename string) error {
	path, err := s.validatedBackupPath(filename)
	if err != nil {
		return err
	}
	return os.Remove(path)
}

func (s *Store) GetBackupPath(filename string) (string, error) {
	path, err := s.validatedBackupPath(filename)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("Backup nicht gefunden")
	}
	return path, nil
}

func (s *Store) RestoreBackup(filename, dbPath string) error {
	backupPath, err := s.GetBackupPath(filename)
	if err != nil {
		return err
	}

	settings, _ := s.GetAppSettings()
	if err := os.MkdirAll(settings.BackupPath, 0755); err != nil {
		return fmt.Errorf("Sicherheitsbackup-Verzeichnis: %w", err)
	}

	// Safety backup before restoring
	safetyName := fmt.Sprintf("vor_wiederherstellung_%s.db", time.Now().Format("2006-01-02_15-04-05"))
	safetyPath := filepath.Join(settings.BackupPath, safetyName)

	if _, err := copyFile(dbPath, safetyPath); err != nil {
		return fmt.Errorf("Sicherheitsbackup erstellen: %w", err)
	}

	// Restore: copy backup over current DB
	if _, err := copyFile(backupPath, dbPath); err != nil {
		return fmt.Errorf("Backup wiederherstellen: %w", err)
	}

	return nil
}

func (s *Store) rotateBackups(settings AppSettings) {
	backups, err := s.ListBackups()
	if err != nil {
		return
	}

	maxCount := settings.BackupMaxCount
	if maxCount < 1 {
		maxCount = DefaultBackupMaxCount
	}
	if len(backups) <= maxCount {
		return
	}

	for _, b := range backups[maxCount:] {
		os.Remove(b.Path)
	}
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
