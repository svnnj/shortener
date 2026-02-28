package service

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

var (
	ErrNilConnection = errors.New("database connection is nil")
)

type (
	HealthCheckerService interface {
		Ping(ctx context.Context) error
	}
	HealthChecker struct {
		db *sql.DB
	}
)

func NewHealthChecker(db *sql.DB) *HealthChecker {
	return &HealthChecker{
		db: db,
	}
}

func (hc *HealthChecker) Ping(ctx context.Context) error {
	if hc.db == nil {
		return ErrNilConnection
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return hc.db.PingContext(ctx)
}
