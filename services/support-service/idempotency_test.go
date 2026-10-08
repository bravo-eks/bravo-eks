package main

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"

	"github.com/lib/pq"
)

func TestIsUniqueViolation(t *testing.T) {
	t.Parallel()

	if isUniqueViolation(nil) {
		t.Fatal("nil error should not be a unique violation")
	}

	if isUniqueViolation(errors.New("store message")) {
		t.Fatal("generic error should not be a unique violation")
	}

	if !isUniqueViolation(&pq.Error{Code: "23505"}) {
		t.Fatal("postgres 23505 should be treated as a unique violation")
	}

	if isUniqueViolation(&pq.Error{Code: "23503"}) {
		t.Fatal("foreign key violations should not be treated as duplicates")
	}
}

func TestParseEmailUsesStableMessageID(t *testing.T) {
	t.Parallel()

	raw := []byte("From: Customer <customer@example.com>\r\n" +
		"To: support@bravoorders.com\r\n" +
		"Subject: Need help with an order\r\n" +
		"Message-ID: <abc-123@mail.example.com>\r\n" +
		"\r\n" +
		"The parcel has not arrived.\r\n")

	email, err := parseEmail(raw)
	if err != nil {
		t.Fatalf("parseEmail: %v", err)
	}

	if email.MessageID != "<abc-123@mail.example.com>" {
		t.Fatalf("MessageID = %q, want %q", email.MessageID, "<abc-123@mail.example.com>")
	}
	if email.From != "customer@example.com" {
		t.Fatalf("From = %q, want customer@example.com", email.From)
	}
}

func TestStoreInboundEmailIsIdempotent(t *testing.T) {
	databaseURL := os.Getenv("SUPPORT_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("SUPPORT_TEST_DATABASE_URL is not set")
	}

	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("ping database: %v", err)
	}

	app := &App{db: db}
	if err := app.migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	t.Cleanup(func() {
		_, _ = db.ExecContext(ctx, `TRUNCATE support.messages, support.tickets RESTART IDENTITY CASCADE`)
	})

	email := Email{
		MessageID: "<ticket-1@mail.example.com>",
		From:      "customer@example.com",
		To:        "support@bravoorders.com",
		Subject:   "Need help",
		Body:      "The parcel has not arrived.",
	}
	s3Key := "incoming/abc-123.eml"

	if err := app.storeInboundEmail(ctx, email, s3Key); err != nil {
		t.Fatalf("first store: %v", err)
	}
	if err := app.storeInboundEmail(ctx, email, s3Key); err != nil {
		t.Fatalf("duplicate s3 key store: %v", err)
	}
	if err := app.storeInboundEmail(ctx, email, "incoming/forwarded-copy.eml"); err != nil {
		t.Fatalf("duplicate message-id store: %v", err)
	}

	var ticketCount, messageCount int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM support.tickets`).Scan(&ticketCount); err != nil {
		t.Fatalf("count tickets: %v", err)
	}
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM support.messages`).Scan(&messageCount); err != nil {
		t.Fatalf("count messages: %v", err)
	}

	if ticketCount != 1 {
		t.Fatalf("ticket count = %d, want 1", ticketCount)
	}
	if messageCount != 1 {
		t.Fatalf("message count = %d, want 1", messageCount)
	}
}
