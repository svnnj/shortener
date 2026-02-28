package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

type urlStoreDB struct {
	db *sql.DB
}

func NewURLStoreDB(db *sql.DB) *urlStoreDB {
	return &urlStoreDB{
		db: db,
	}
}

func (s *urlStoreDB) Get(ctx context.Context, token string) (string, error) {
	row := s.db.QueryRowContext(ctx, "SELECT url_ FROM t_short_url WHERE token = $1", token)
	var url string

	if err := row.Scan(&url); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("getting url for token (%s): %w", token, ErrNotFound)
		}
		return "", fmt.Errorf("getting url for token (%s): %w", token, err)
	}

	return url, nil
}

func (s *urlStoreDB) Set(ctx context.Context, rec URLRec) error {
	if _, err := s.db.ExecContext(
		ctx,
		"INSERT INTO t_short_url (token, url_) values ($1, $2)",
		rec.Token,
		rec.OriginalURL,
	); err != nil {
		return fmt.Errorf("saving url with token %q: %w", rec.Token, err)
	}

	return nil
}

func (s *urlStoreDB) SetBatch(ctx context.Context, urlRecs []URLRec) error {

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begining transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, "INSERT INTO t_short_url (token, url_) values ($1, $2)")
	if err != nil {
		return fmt.Errorf("prepating stmt: %w", err)
	}

	for _, v := range urlRecs {
		_, err := stmt.ExecContext(ctx, v.Token, v.OriginalURL)
		if err != nil {
			return fmt.Errorf("executing insert for url %s (token %q): %w", v.OriginalURL, v.Token, err)
		}
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("committing batch insert: %w", err)
	}
	return nil
}
