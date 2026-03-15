package models

import (
	"database/sql"
	"log/slog"
)

type Store struct {
	DB *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{DB: db}
}

type Transaction struct {
	Tx *sql.Tx
}

func (s *Store) Begin() (*Transaction, error) {
	tx, err := s.DB.Begin()
	if err != nil {
		slog.Error("Failed to begin transaction", "error", err)
		return nil, err
	}
	slog.Debug("Transaction started")
	return &Transaction{Tx: tx}, nil
}

func (t *Transaction) Commit() error {
	err := t.Tx.Commit()
	if err != nil {
		slog.Error("Failed to commit transaction", "error", err)
	} else {
		slog.Debug("Transaction committed")
	}
	return err
}

func (t *Transaction) Rollback() error {
	err := t.Tx.Rollback()
	if err != nil {
		slog.Error("Failed to rollback transaction", "error", err)
	} else {
		slog.Debug("Transaction rolled back")
	}
	return err
}
