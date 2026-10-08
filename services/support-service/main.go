package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net"
	"net/http"
	"net/mail"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/lib/pq"
)

type Config struct {
	Port        string
	DatabaseURL string
	AWSRegion   string
	S3Bucket    string
	S3Prefix    string
	SQSQueueURL string
	SupportFrom string
}

type App struct {
	cfg Config
	db  *sql.DB
	s3  *s3.Client
	sqs *sqs.Client
	ses *sesv2.Client
}

type S3Event struct {
	Records []struct {
		S3 struct {
			Bucket struct {
				Name string `json:"name"`
			} `json:"bucket"`
			Object struct {
				Key string `json:"key"`
			} `json:"object"`
		} `json:"s3"`
	} `json:"Records"`
}

type Email struct {
	MessageID string
	From      string
	To        string
	Subject   string
	Body      string
}

func main() {
	cfg := loadConfig()

	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database open failed: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("database connection failed: %v", err)
	}

	appCfg, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithRegion(cfg.AWSRegion),
	)
	if err != nil {
		log.Fatalf("AWS config failed: %v", err)
	}

	app := &App{
		cfg: cfg,
		db:  db,
		s3:  s3.NewFromConfig(appCfg),
		sqs: sqs.NewFromConfig(appCfg),
		ses: sesv2.NewFromConfig(appCfg),
	}

	if strings.EqualFold(getEnv("RUN_MIGRATIONS", "false"), "true") {
		if err := app.migrate(context.Background()); err != nil {
			log.Fatalf("migration failed: %v", err)
		}

		if strings.EqualFold(getEnv("MIGRATIONS_ONLY", "false"), "true") {
			log.Println("Migration-only mode complete")
			return
		}
	}

	go app.consumeSQS(context.Background())

	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if err := db.PingContext(r.Context()); err != nil {
			http.Error(w, "database unavailable", http.StatusServiceUnavailable)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	mux.HandleFunc("/livez", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("support-service listening on :%s", cfg.Port)

	if err := server.ListenAndServe(); err != nil &&
		!errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server failed: %v", err)
	}
}

func loadConfig() Config {
	return Config{
		Port:        getEnv("PORT", "8087"),
		DatabaseURL: mustEnv("DATABASE_URL"),
		AWSRegion:   getEnv("AWS_REGION", "eu-west-2"),
		S3Bucket:    mustEnv("SUPPORT_S3_BUCKET"),
		S3Prefix:    getEnv("SUPPORT_S3_PREFIX", "incoming/"),
		SQSQueueURL: mustEnv("SUPPORT_SQS_QUEUE_URL"),
		SupportFrom: getEnv("SUPPORT_FROM_EMAIL", "support@bravoorders.com"),
	}
}

func (a *App) migrate(ctx context.Context) error {
	migrations := []string{

		`CREATE TABLE IF NOT EXISTS support.tickets (
			id BIGSERIAL PRIMARY KEY,
			customer_email TEXT NOT NULL,
			subject TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'open'
				CHECK (status IN ('open', 'pending', 'closed')),
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			last_message_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,

		`CREATE TABLE IF NOT EXISTS support.messages (
			id BIGSERIAL PRIMARY KEY,
			ticket_id BIGINT NOT NULL REFERENCES support.tickets(id) ON DELETE CASCADE,
			direction TEXT NOT NULL
				CHECK (direction IN ('inbound', 'outbound')),
			message_id TEXT,
			from_email TEXT NOT NULL,
			to_email TEXT NOT NULL,
			subject TEXT NOT NULL DEFAULT '',
			body_text TEXT NOT NULL DEFAULT '',
			s3_key TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,

		`CREATE INDEX IF NOT EXISTS idx_support_tickets_customer_email
			ON support.tickets(customer_email)`,

		`CREATE INDEX IF NOT EXISTS idx_support_tickets_status
			ON support.tickets(status)`,

		`CREATE INDEX IF NOT EXISTS idx_support_tickets_last_message
			ON support.tickets(last_message_at DESC)`,

		`CREATE INDEX IF NOT EXISTS idx_support_messages_ticket
			ON support.messages(ticket_id)`,

		`CREATE INDEX IF NOT EXISTS idx_support_messages_message_id
			ON support.messages(message_id)`,

		`CREATE UNIQUE INDEX IF NOT EXISTS idx_support_messages_s3_key
			ON support.messages(s3_key)
			WHERE s3_key IS NOT NULL`,

		`CREATE UNIQUE INDEX IF NOT EXISTS idx_support_messages_message_id_unique
			ON support.messages(message_id)
			WHERE message_id IS NOT NULL AND message_id <> ''`,
	}

	for _, migration := range migrations {
		if _, err := a.db.ExecContext(ctx, migration); err != nil {
			return err
		}
	}

	log.Println("Support database migrations complete")
	return nil
}

func (a *App) consumeSQS(ctx context.Context) {
	log.Println("Support SQS consumer started")

	for {
		result, err := a.sqs.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(a.cfg.SQSQueueURL),
			MaxNumberOfMessages: 10,
			WaitTimeSeconds:     20,
			VisibilityTimeout:   60,
		})
		if err != nil {
			log.Printf("SQS receive failed: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		for _, message := range result.Messages {
			if err := a.processSQSMessage(ctx, aws.ToString(message.Body)); err != nil {
				log.Printf("support message processing failed: %v", err)
				continue
			}

			if _, err := a.sqs.DeleteMessage(ctx, &sqs.DeleteMessageInput{
				QueueUrl:      aws.String(a.cfg.SQSQueueURL),
				ReceiptHandle: message.ReceiptHandle,
			}); err != nil {
				log.Printf("SQS delete failed: %v", err)
			}
		}
	}
}

func (a *App) processSQSMessage(ctx context.Context, body string) error {
	var event S3Event

	if err := json.Unmarshal([]byte(body), &event); err != nil {
		return fmt.Errorf("decode S3 event: %w", err)
	}

	for _, record := range event.Records {
		bucket := record.S3.Bucket.Name
		key, err := urlDecodeS3Key(record.S3.Object.Key)
		if err != nil {
			return err
		}

		if bucket == "" {
			bucket = a.cfg.S3Bucket
		}

		if !strings.HasPrefix(key, a.cfg.S3Prefix) {
			log.Printf("ignoring object outside prefix: %s", key)
			continue
		}

		if err := a.processS3Email(ctx, bucket, key); err != nil {
			return err
		}
	}

	return nil
}

func (a *App) processS3Email(ctx context.Context, bucket, key string) error {
	result, err := a.s3.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("get S3 object %s: %w", key, err)
	}
	defer result.Body.Close()

	raw, err := io.ReadAll(result.Body)
	if err != nil {
		return fmt.Errorf("read S3 object %s: %w", key, err)
	}

	email, err := parseEmail(raw)
	if err != nil {
		return fmt.Errorf("parse email %s: %w", key, err)
	}

	return a.storeInboundEmail(ctx, email, key)
}

func (a *App) storeInboundEmail(
	ctx context.Context,
	email Email,
	s3Key string,
) error {
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	duplicate, err := inboundEmailAlreadyStored(ctx, tx, s3Key, email.MessageID)
	if err != nil {
		return err
	}
	if duplicate {
		log.Printf("skipping duplicate inbound email s3_key=%s message_id=%s", s3Key, email.MessageID)
		return nil
	}

	var ticketID int64

	err = tx.QueryRowContext(ctx, `
		SELECT id
		FROM support.tickets
		WHERE customer_email = $1
		  AND status <> 'closed'
		ORDER BY updated_at DESC
		LIMIT 1
	`, email.From).Scan(&ticketID)

	if err == sql.ErrNoRows {
		err = tx.QueryRowContext(ctx, `
			INSERT INTO support.tickets (
				customer_email,
				subject
			)
			VALUES ($1, $2)
			RETURNING id
		`, email.From, email.Subject).Scan(&ticketID)

		if err != nil {
			return fmt.Errorf("create ticket: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("find ticket: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO support.messages (
			ticket_id,
			direction,
			message_id,
			from_email,
			to_email,
			subject,
			body_text,
			s3_key
		)
		VALUES ($1, 'inbound', NULLIF($2, ''), $3, $4, $5, $6, $7)
	`, ticketID,
		email.MessageID,
		email.From,
		email.To,
		email.Subject,
		email.Body,
		s3Key,
	)
	if err != nil {
		if isUniqueViolation(err) {
			log.Printf("skipping concurrent duplicate inbound email s3_key=%s message_id=%s", s3Key, email.MessageID)
			return nil
		}
		return fmt.Errorf("store message: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE support.tickets
		SET
			updated_at = NOW(),
			last_message_at = NOW(),
			subject = CASE
				WHEN subject = '' THEN $2
				ELSE subject
			END
		WHERE id = $1
	`, ticketID, email.Subject)
	if err != nil {
		return fmt.Errorf("update ticket: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	log.Printf(
		"stored inbound support email ticket=%d from=%s subject=%q",
		ticketID,
		email.From,
		email.Subject,
	)

	return nil
}

func inboundEmailAlreadyStored(ctx context.Context, tx *sql.Tx, s3Key, messageID string) (bool, error) {
	var ticketID int64

	err := tx.QueryRowContext(ctx, `
		SELECT ticket_id
		FROM support.messages
		WHERE s3_key = $1
		   OR (NULLIF($2, '') IS NOT NULL AND message_id = $2)
		LIMIT 1
	`, s3Key, messageID).Scan(&ticketID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check duplicate inbound email: %w", err)
	}

	return true, nil
}

func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr.Code == "23505"
	}
	return false
}

func parseEmail(raw []byte) (Email, error) {
	msg, err := mail.ReadMessage(strings.NewReader(string(raw)))
	if err != nil {
		return Email{}, err
	}

	from := firstAddress(msg.Header.Get("From"))
	to := firstAddress(msg.Header.Get("To"))

	if from == "" {
		return Email{}, fmt.Errorf("missing From header")
	}

	body, err := readBody(msg)
	if err != nil {
		return Email{}, err
	}

	return Email{
		MessageID: strings.TrimSpace(msg.Header.Get("Message-ID")),
		From:      from,
		To:        to,
		Subject:   strings.TrimSpace(msg.Header.Get("Subject")),
		Body:      strings.TrimSpace(body),
	}, nil
}

func readBody(msg *mail.Message) (string, error) {
	contentType := msg.Header.Get("Content-Type")
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		mediaType = "text/plain"
		params = nil
	}

	if strings.HasPrefix(mediaType, "multipart/") {
		boundary := params["boundary"]
		if boundary == "" {
			return "", fmt.Errorf("multipart email missing boundary")
		}

		mr := multipart.NewReader(msg.Body, boundary)

		for {
			part, err := mr.NextPart()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				return "", err
			}

			partContentType := part.Header.Get("Content-Type")
			partMediaType, _, _ := mime.ParseMediaType(partContentType)

			if strings.EqualFold(partMediaType, "text/plain") ||
				partMediaType == "" {
				data, err := io.ReadAll(part)
				if err != nil {
					return "", err
				}
				return string(data), nil
			}
		}

		return "", nil
	}

	data, err := io.ReadAll(msg.Body)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func firstAddress(value string) string {
	value = strings.TrimSpace(value)

	if value == "" {
		return ""
	}

	addr, err := mail.ParseAddress(value)
	if err == nil {
		return addr.Address
	}

	if strings.Contains(value, ",") {
		return strings.TrimSpace(strings.Split(value, ",")[0])
	}

	return value
}

func urlDecodeS3Key(key string) (string, error) {
	key = strings.ReplaceAll(key, "+", " ")

	decoded, err := pathUnescape(key)
	if err != nil {
		return "", fmt.Errorf("decode S3 key: %w", err)
	}

	return decoded, nil
}

func pathUnescape(value string) (string, error) {
	var result strings.Builder

	for i := 0; i < len(value); i++ {
		if value[i] != '%' {
			result.WriteByte(value[i])
			continue
		}

		if i+2 >= len(value) {
			return "", fmt.Errorf("invalid URL encoding")
		}

		var b byte
		if _, err := fmt.Sscanf(value[i+1:i+3], "%02x", &b); err != nil {
			return "", err
		}

		result.WriteByte(b)
		i += 2
	}

	return result.String(), nil
}

func mustEnv(key string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		log.Fatalf("%s is required", key)
	}
	return value
}

func getEnv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

// Keep net imported as part of the service's future HTTP/network validation
// surface and prevent accidental unused import churn during service expansion.
var _ net.IP
