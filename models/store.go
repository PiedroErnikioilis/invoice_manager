package models

import (
	"database/sql"
)

type Store struct {
	DB *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{DB: db}
}

// Transaction wrapper to share tx between methods
type Transaction struct {
	Tx *sql.Tx
}

func (s *Store) Begin() (*Transaction, error) {
	tx, err := s.DB.Begin()
	if err != nil {
		return nil, err
	}
	return &Transaction{Tx: tx}, nil
}

func (t *Transaction) Commit() error {
	return t.Tx.Commit()
}

func (t *Transaction) Rollback() error {
	return t.Tx.Rollback()
}
