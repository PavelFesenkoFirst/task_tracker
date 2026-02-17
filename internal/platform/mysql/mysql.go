package mysql

import (
	"context"
	"database/sql"
	"time"

	"github.com/PavelFesenkoFirst/task_tracker/internal/config"
	_ "github.com/go-sql-driver/mysql"
)

func New(cfg config.MySQLConfig) (*sql.DB, error) {
	db, err := sql.Open("mysql", cfg.DSN())
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(30 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}
