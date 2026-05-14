package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/viper"
	"github.com/thatsbass/veil/pkg/utils"
)

const (
	testEmail = "test@veil.dev"
	testKey   = "vl_live_testkey1234567890abcdefghijklmnopqrstuvw"
	testName  = "test-key"
)

func main() {
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()
	_ = viper.ReadInConfig()

	dsn := viper.GetString("DATABASE_URL")
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "DATABASE_URL non défini")
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := pgxpool.New(ctx, dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "connexion DB : %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := seed(ctx, db); err != nil {
		fmt.Fprintf(os.Stderr, "seed : %v\n", err)
		os.Exit(1)
	}
}

func seed(ctx context.Context, db *pgxpool.Pool) error {
	// Upsert user
	var userID string
	err := db.QueryRow(ctx, `
		INSERT INTO users (email, plan)
		VALUES ($1, 'free')
		ON CONFLICT (email) DO UPDATE SET email = EXCLUDED.email
		RETURNING id
	`, testEmail).Scan(&userID)
	if err != nil {
		return fmt.Errorf("upsert user : %w", err)
	}

	// Upsert api key
	hash := utils.HashAPIKey(testKey)
	_, err = db.Exec(ctx, `
		INSERT INTO api_keys (user_id, key_hash, name)
		VALUES ($1, $2, $3)
		ON CONFLICT (key_hash) DO NOTHING
	`, userID, hash, testName)
	if err != nil {
		return fmt.Errorf("upsert api_key : %w", err)
	}

	fmt.Println()
	fmt.Println("✓ Seed terminé")
	fmt.Println()
	fmt.Printf("  Email    : %s\n", testEmail)
	fmt.Printf("  Clé API  : %s\n", testKey)
	fmt.Println()
	fmt.Printf("  Swagger  → Authorize → Bearer %s\n", testKey)
	fmt.Println()
	return nil
}
