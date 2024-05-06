package database

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog/log"
)

type Sqlite struct {
	dbName string
	dsn    string
	conn   *sqlx.DB

	mx *sync.RWMutex
}

func ConnectSqlite(dbName string) *Sqlite {
	return &Sqlite{
		dbName: dbName,
		dsn:    "sqlite3",
		mx:     &sync.RWMutex{},
	}
}

func (sqlite *Sqlite) Connect(ctx context.Context) {
	db, err := sqlx.Connect(sqlite.dsn, sqlite.dbName)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to sqlite")
	}

	sqlite.mx.Lock()
	defer sqlite.mx.Unlock()

	sqlite.conn = db

	if err := sqlite.conn.Ping(); err != nil {
		log.Fatal().Err(err).Msg("failed to reach to sqlite")
	}
}

const MigrationDir = "./migrations/sqlite"

func (sqlite *Sqlite) Setup(ctx context.Context) error {
	var err error

	schemaFile := fmt.Sprintf("%s/schema.up.sql", MigrationDir)

	data, err := os.ReadFile(schemaFile)
	if err != nil {
		return err
	}

	tx, err := sqlite.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			log.Info().Msg("rolling back schema changes")
			tx.Rollback()
		}
	}()

	stmts := strings.Split(string(data), "---")

	for _, stmt := range stmts {
		_, err = tx.ExecContext(ctx, strings.TrimSpace(stmt))
		if err != nil {
			log.Error().Err(err).Msg("failed to setup db")
			return err
		}
	}

	err = tx.Commit()
	return err
}
