package repository

import (
	"context"
	"database/sql"
	"fmt"
)

type kvStoreDB struct {
	db *sql.DB
}

func NewKVStoreDB(db *sql.DB) *kvStoreDB {
	return &kvStoreDB{
		db: db,
	}
}

func (s *kvStoreDB) Get(ctx context.Context, key string) (string, error) {
	row := s.db.QueryRowContext(ctx, "SELECT val FROM t_shortener WHERE key = $1", key)
	var val string
	err := row.Scan(&val)
	if err != nil {
		return "", fmt.Errorf("getting url: %w", err)
	}

	return val, nil
}

func (s *kvStoreDB) Set(ctx context.Context, key string, val string) error {
	_, err := s.db.ExecContext(ctx, "INSERT INTO t_shortener (key, val) values ($1, $2)", key, val)
	if err != nil {
		return fmt.Errorf("saving url in db: %w", err)
	}

	return nil
}

func (s *kvStoreDB) SetBatch(ctx context.Context, kvs map[string]string) error {

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begining transaction: %w", err)
	}

	stmt, err := tx.PrepareContext(ctx, "INSERT INTO t_shortener (key, val) values ($1, $2)")
	if err != nil {
		return fmt.Errorf("prepating stmt: %w", err)
	}

	for k, v := range kvs {
		_, err := stmt.ExecContext(ctx, k, v)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("prepating stmt: %w", err)
		}
	}
	tx.Commit()

	return nil
}
