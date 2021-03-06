package storage

import (
	"database/sql"
	"fmt"

	"github.com/lib/pq"
	"github.com/opentable/sous/util/logging"
)

type (
	// The PostgresStateManager provides the StateManager interface by
	// reading/writing from a postgres database.
	PostgresStateManager struct {
		db  *sql.DB
		log logging.LogSink
	}

	// A PostgresConfig describes how to connect to a postgres database
	PostgresConfig struct {
		DBName   string `env:"SOUS_PG_DBNAME"`
		User     string `env:"SOUS_PG_USER"`
		Password string `env:"SOUS_PG_PASSWORD"`
		Host     string `env:"SOUS_PG_HOST"`
		Port     string `env:"SOUS_PG_PORT"`
		SSL      bool   `env:"SOUS_PG_SSL"`
	}
)

// NewPostgresStateManager creates a new PostgresStateManager.
func NewPostgresStateManager(db *sql.DB, log logging.LogSink) *PostgresStateManager {
	return &PostgresStateManager{db: db, log: log}
}

func (c PostgresConfig) connStr() string {
	sslmode := "enable"
	if !c.SSL {
		sslmode = "disable"
	}
	conn := fmt.Sprintf("dbname=%s host=%s port=%s sslmode=%s", c.DBName, c.Host, c.Port, sslmode)
	if c.User != "" {
		conn = fmt.Sprintf("%s user=%s", conn, c.User)
	}
	if c.Password != "" {
		conn = fmt.Sprintf("%s password=%s", conn, c.Password)
	}
	return conn
}

// DB returns a database connection based on this config
func (c PostgresConfig) DB() (*sql.DB, error) {
	db, err := sql.Open("postgres", c.connStr())
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}

func isNoDBError(err error) bool {
	pqerr, is := err.(*pq.Error)
	if !is {
		return false
	}
	return pqerr.Code == "3D000" // invalid_catalog_name per https://www.postgresql.org/docs/current/static/errcodes-appendix.html
}
