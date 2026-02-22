package service

import (
	"context"
	"database/sql"
)

type HealthCheckerService interface {
	PingDB(ctx context.Context) error
}

type HealthChecker struct {
	db *sql.DB
}

func NewHealthChecker(db *sql.DB) *HealthChecker {
	return &HealthChecker{
		db: db,
	}
}

func (hc *HealthChecker) PingDB(ctx context.Context) error {
	return hc.db.PingContext(ctx)
}
