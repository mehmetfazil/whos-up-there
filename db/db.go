package db

import (
	"context"
	"log"

	"github.com/jackc/pgx/v4/pgxpool"
)

var Pool *pgxpool.Pool

func Init(connString string) error {
	var err error
	Pool, err = pgxpool.Connect(context.Background(), connString)
	if err != nil {
		return err
	}
	return nil
}

func Close() {
	if Pool != nil {
		Pool.Close()
		log.Println("Database connection closed")
	}
}
