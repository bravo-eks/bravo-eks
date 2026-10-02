package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

const (
	defaultHTTPTimeout = 10 * time.Second
	defaultWaitTime    = 20
	defaultMaxMessages = 10
)

type Event struct {
	Type      string                 `json:"type"`
	Payload   map[string]interface{} `json:"payload"`
	Timestamp string                 `json:"timestamp"`
}

type Services struct {
	Inventory    string
	Payment      string
	Notification string
	Shipping     string
	Order        string
}

type Worker struct {
	sqsClient  *sqs.Client
	queueURL   string
	services   Services
	httpClient *http.Client
}

func main() {
	queueURL := strings.TrimSpace(os.Getenv("SQS_QUEUE_URL"))
	if queueURL == "" {
		log.Fatal("SQS_QUEUE_URL is required")
	}

	awsConfig, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Fatalf("failed to load AWS configuration: %v", err)
	}

	sqsClient := sqs.NewFromConfig(awsConfig, func(options *sqs.Options) {
		if endpoint := strings.TrimSpace(os.Getenv("SQS_ENDPOINT_URL")); endpoint != "" {
			options.BaseEndpoint = aws.String(endpoint)
		}
	})

	httpTimeout := getDurationEnv("WORKER_HTTP_TIMEOUT", defaultHTTPTimeout)

	worker := &Worker{
		sqsClient: sqsClient,
		queueURL:  queueURL,
		services: Services{
			Inventory:    getEnv("INVENTORY_SERVICE_URL", "http://inventory-service:8082"),
			Payment:      getEnv("PAYMENT_SERVICE_URL", "http://payment-service:8083"),
			Notification: getEnv("NOTIFICATION_SERVICE_URL", "http://notification-service:8084"),
			Shipping:     getEnv("SHIPPING_SERVICE_URL", "http://shipping-service:8085"),
			Order:        getEnv("ORDER_SERVICE_URL", "http://order-service:8081"),
		},
		httpClient: &http.Client{
			Timeout: httpTimeout,
		},
	}

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer cancel()

	healthServer := startHealthServer()

	log.Println("Worker started, polling SQS for events...")

	if err := worker.pollAndProcess(ctx); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatalf("worker stopped with error: %v", err)
	}

	log.Println("Worker stopped")

	shutdownCtx, shutdownCancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer shutdownCancel()

	if err := healthServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("health server shutdown error: %v", err)
	}
}

func startHealthServer() *http.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/livez", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":  "ok",
			"service": "worker",
		})
	})

	port := getEnv("HEALTH_PORT", "8090")

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	go func() {
		log.Printf("Worker health check on :%s", port)

		if err := server.ListenAndServe(); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			log.Printf("health server error: %v", err)
		}
	}()

	return server
}

func (w *Worker) pollAndProcess(ctx context.Context) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		messages, err := w.receiveMessages(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return err
			}

			log.Printf("failed to receive SQS messages: %v", err)

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(2 * time.Second):
			}

			continue
		}

		for _, message := range messages {
			if err := ctx.Err(); err != nil {
				return err
			}

			if message.Body == nil {
				log.Printf("received SQS message without body; leaving message for retry")
				continue
			}

			if message.ReceiptHandle == nil ||
				strings.TrimSpace(aws.ToString(message.ReceiptHandle)) == "" {
				log.Printf("received SQS message without receipt handle; leaving message for retry")
				continue
			}

			if err := w.processMessage(ctx, message); err != nil {
				log.Printf(
					"failed to process SQS message %s: %v; leaving message for retry/DLQ",
					aws.ToString(message.MessageId),
					err,
				)

				continue
			}

			if err := w.deleteMessage(ctx, aws.ToString(message.ReceiptHandle)); err != nil {
				if errors.Is(err, context.Canceled) {
					return err
				}

				log.Printf(
					"failed to delete successfully processed message %s: %v",
					aws.ToString(message.MessageId),
					err,
				)

				// The message has already been processed successfully.
				// If deletion fails, SQS will make it visible again after
				// the visibility timeout. This is expected at-least-once behaviour.
				continue
			}

			log.Printf(
				"successfully processed and deleted message %s",
				aws.ToString(message.MessageId),
			)
		}
	}
}

func (w *Worker) receiveMessages(ctx context.Context) ([]types.Message, error) {
	result, err := w.sqsClient.ReceiveMessage(
		ctx,
		&sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(w.queueURL),
			MaxNumberOfMessages: defaultMaxMessages,
			WaitTimeSeconds:     defaultWaitTime,
			VisibilityTimeout:   getInt32Env("SQS_VISIBILITY_TIMEOUT", 60),
			MessageAttributeNames: []string{
				"All",
			},
			MessageSystemAttributeNames: []types.MessageSystemAttributeName{
				types.MessageSystemAttributeNameApproximateReceiveCount,
			},
		},
	)

	if err != nil {
		return nil, fmt.Errorf("receive SQS messages: %w", err)
	}

	return result.Messages, nil
}

func (w *Worker) deleteMessage(ctx context.Context, receiptHandle string) error {
	if strings.TrimSpace(receiptHandle) == "" {
		return errors.New("receipt handle is required")
	}

	_, err := w.sqsClient.DeleteMessage(
		ctx,
		&sqs.DeleteMessageInput{
			QueueUrl:      aws.String(w.queueURL),
			ReceiptHandle: aws.String(receiptHandle),
		},
	)

	if err != nil {
		return fmt.Errorf("delete SQS message: %w", err)
	}

	return nil
}

func (w *Worker) processMessage(
	ctx context.Context,
	message types.Message,
) error {
	if message.Body == nil {
		return errors.New("message body is missing")
	}

	var event Event

	if err := json.Unmarshal(
		[]byte(aws.ToString(message.Body)),
		&event,
	); err != nil {
		return fmt.Errorf("malformed event JSON: %w", err)
	}

	event.Type = strings.TrimSpace(event.Type)

	if event.Type == "" {
		return errors.New("event type is required")
	}

	if event.Payload == nil {
		return fmt.Errorf("event %s payload is required", event.Type)
	}

	log.Printf(
		"Processing event: type=%s message_id=%s",
		event.Type,
		aws.ToString(message.MessageId),
	)

	if err := w.handleEvent(ctx, event); err != nil {
		return fmt.Errorf("handle %s: %w", event.Type, err)
	}

	return nil
}

func (w *Worker) handleEvent(
	ctx context.Context,
	event Event,
) error {
	switch event.Type {
	case "order.created":
		return w.handleOrderCreated(ctx, event.Payload)

	case "order.status_changed":
		return w.handleOrderStatusChanged(ctx, event.Payload)

	case "payment.completed":
		return w.handlePaymentCompleted(ctx, event.Payload)

	case "payment.failed":
		return w.handlePaymentFailed(ctx, event.Payload)

	case "shipment.created":
		return w.handleShipmentCreated(ctx, event.Payload)

	case "shipment.delivered":
		return w.handleShipmentDelivered(ctx, event.Payload)

	default:
		return fmt.Errorf("unsupported event type: %q", event.Type)
	}
}

func (w *Worker) handleOrderCreated(
	ctx context.Context,
	payload map[string]interface{},
) error {
	orderID, err := requiredInt(payload, "order_id")
	if err != nil {
		return err
	}

	customerID, err := requiredString(payload, "customer_id")
	if err != nil {
		return err
	}

	items, ok := payload["items"]
	if !ok || items == nil {
		return errors.New("order.created requires items")
	}

	total, err := requiredNumber(payload, "total")
	if err != nil {
		return err
	}

	currency := stringValue(payload, "currency", "GBP")

	log.Printf("order.created: reserving inventory for order %d", orderID)

	reservePayload := map[string]interface{}{
		"order_id": orderID,
		"items":    items,
	}

	if err := w.postJSON(
		ctx,
		w.services.Inventory,
		"/reserve",
		reservePayload,
		nil,
	); err != nil {
		return fmt.Errorf("reserve inventory: %w", err)
	}

	log.Printf("order.created: processing payment for order %d", orderID)

	chargePayload := map[string]interface{}{
		"order_id":    orderID,
		"customer_id": customerID,
		"amount":      total,
		"currency":    currency,
		"method":      "worker",
	}

	if err := w.postJSON(
		ctx,
		w.services.Payment,
		"/charge",
		chargePayload,
		nil,
	); err != nil {
		log.Printf(
			"payment failed for order %d; releasing inventory reservation",
			orderID,
		)

		releaseErr := w.postJSON(
			ctx,
			w.services.Inventory,
			"/release",
			map[string]interface{}{
				"order_id": orderID,
			},
			nil,
		)

		if releaseErr != nil {
			return fmt.Errorf(
				"payment failed: %v; inventory release also failed: %w",
				err,
				releaseErr,
			)
		}

		return fmt.Errorf("process payment: %w", err)
	}

	log.Printf("order.created: sending confirmation for order %d", orderID)

	notificationPayload := map[string]interface{}{
		"recipient": customerID,
		"channel":   "email",
		"template":  "order_confirmed",
		"data": map[string]interface{}{
			"OrderID":     orderID,
			"CustomerName": customerID,
			"Currency":    currency,
			"Total":       total,
		},
	}

	if err := w.postJSON(
		ctx,
		w.services.Notification,
		"/send",
		notificationPayload,
		nil,
	); err != nil {
		return fmt.Errorf("send order confirmation: %w", err)
	}

	log.Printf("order.created: confirming order %d", orderID)

	if err := w.putJSON(
		ctx,
		w.services.Order,
		"/status",
		map[string]interface{}{
			"order_id":   orderID,
			"new_status": "confirmed",
		},
		nil,
	); err != nil {
		return fmt.Errorf("confirm order: %w", err)
	}

	return nil
}

func (w *Worker) handleOrderStatusChanged(
	ctx context.Context,
	payload map[string]interface{},
) error {
	orderID, err := requiredInt(payload, "order_id")
	if err != nil {
		return err
	}

	newStatus, err := requiredString(payload, "new_status")
	if err != nil {
		return err
	}

	switch newStatus {
	case "processing":
		return w.createShipment(ctx, payload, orderID)

	case "shipped":
		trackingNumber := stringValue(payload, "tracking_number", "")

		return w.sendNotification(
			ctx,
			payload,
			"order_shipped",
			map[string]interface{}{
				"OrderID":        orderID,
				"TrackingNumber": trackingNumber,
			},
		)

	case "delivered":
		return w.sendNotification(
			ctx,
			payload,
			"order_delivered",
			map[string]interface{}{
				"OrderID": orderID,
			},
		)

	case "cancelled":
		if err := w.postJSON(
			ctx,
			w.services.Inventory,
			"/release",
			map[string]interface{}{
				"order_id": orderID,
			},
			nil,
		); err != nil {
			return fmt.Errorf("release inventory: %w", err)
		}

		paymentID := stringValue(payload, "payment_id", "")

		if paymentID != "" {
			amount, err := optionalNumber(payload, "amount")
			if err != nil {
				return fmt.Errorf("invalid refund amount: %w", err)
			}

			if amount > 0 {
				if err := w.postJSON(
					ctx,
					w.services.Payment,
					"/refund",
					map[string]interface{}{
						"payment_id": paymentID,
						"amount":     amount,
						"reason":     "order cancelled",
					},
					nil,
				); err != nil {
					return fmt.Errorf("refund payment: %w", err)
				}
			}
		}

		return nil

	default:
		return fmt.Errorf(
			"unsupported order status change: %q",
			newStatus,
		)
	}
}

func (w *Worker) createShipment(
	ctx context.Context,
	payload map[string]interface{},
	orderID int,
) error {
	shipmentPayload := map[string]interface{}{
		"order_id":       orderID,
		"carrier":        stringValue(payload, "carrier", "royal_mail"),
		"recipient_name": stringValue(payload, "recipient_name", ""),
		"address_line1":  stringValue(payload, "address_line1", ""),
		"address_line2":  stringValue(payload, "address_line2", ""),
		"city":           stringValue(payload, "city", ""),
		"postcode":       stringValue(payload, "postcode", ""),
		"country":        stringValue(payload, "country", "GB"),
	}

	if weight, ok := payload["weight_kg"]; ok {
		shipmentPayload["weight_kg"] = weight
	}

	if err := w.postJSON(
		ctx,
		w.services.Shipping,
		"/shipments",
		shipmentPayload,
		nil,
	); err != nil {
		return fmt.Errorf("create shipment for order %d: %w", orderID, err)
	}

	return nil
}

func (w *Worker) handlePaymentCompleted(
	ctx context.Context,
	payload map[string]interface{},
) error {
	orderID, err := requiredInt(payload, "order_id")
	if err != nil {
		return err
	}

	log.Printf(
		"payment.completed: confirming order %d",
		orderID,
	)

	return w.putJSON(
		ctx,
		w.services.Order,
		"/status",
		map[string]interface{}{
			"order_id":   orderID,
			"new_status": "confirmed",
		},
		nil,
	)
}

func (w *Worker) handlePaymentFailed(
	ctx context.Context,
	payload map[string]interface{},
) error {
	orderID, err := requiredInt(payload, "order_id")
	if err != nil {
		return err
	}

	log.Printf(
		"payment.failed: releasing inventory for order %d",
		orderID,
	)

	if err := w.postJSON(
		ctx,
		w.services.Inventory,
		"/release",
		map[string]interface{}{
			"order_id": orderID,
		},
		nil,
	); err != nil {
		return fmt.Errorf("release inventory: %w", err)
	}

	log.Printf(
		"payment.failed: cancelling order %d",
		orderID,
	)

	if err := w.putJSON(
		ctx,
		w.services.Order,
		"/status",
		map[string]interface{}{
			"order_id":   orderID,
			"new_status": "cancelled",
		},
		nil,
	); err != nil {
		return fmt.Errorf("cancel order: %w", err)
	}

	customerID := stringValue(payload, "customer_id", "")

	if customerID != "" {
		if err := w.sendNotification(
			ctx,
			payload,
			"payment_failed",
			map[string]interface{}{
				"OrderID": orderID,
			},
		); err != nil {
			return fmt.Errorf("send payment failure notification: %w", err)
		}
	}

	return nil
}

func (w *Worker) handleShipmentCreated(
	ctx context.Context,
	payload map[string]interface{},
) error {
	orderID, err := requiredInt(payload, "order_id")
	if err != nil {
		return err
	}

	log.Printf(
		"shipment.created: updating order %d to processing",
		orderID,
	)

	return w.putJSON(
		ctx,
		w.services.Order,
		"/status",
		map[string]interface{}{
			"order_id":   orderID,
			"new_status": "processing",
		},
		nil,
	)
}

func (w *Worker) handleShipmentDelivered(
	ctx context.Context,
	payload map[string]interface{},
) error {
	orderID, err := requiredInt(payload, "order_id")
	if err != nil {
		return err
	}

	log.Printf(
		"shipment.delivered: updating order %d to delivered",
		orderID,
	)

	if err := w.putJSON(
		ctx,
		w.services.Order,
		"/status",
		map[string]interface{}{
			"order_id":   orderID,
			"new_status": "delivered",
		},
		nil,
	); err != nil {
		return fmt.Errorf("mark order delivered: %w", err)
	}

	customerID := stringValue(payload, "customer_id", "")

	if customerID != "" {
		if err := w.sendNotification(
			ctx,
			payload,
			"order_delivered",
			map[string]interface{}{
				"OrderID": orderID,
			},
		); err != nil {
			return fmt.Errorf("send delivery notification: %w", err)
		}
	}

	return nil
}

func (w *Worker) sendNotification(
	ctx context.Context,
	payload map[string]interface{},
	template string,
	data map[string]interface{},
) error {
	customerID := stringValue(payload, "customer_id", "")

	if customerID == "" {
		return errors.New("customer_id is required for notification")
	}

	return w.postJSON(
		ctx,
		w.services.Notification,
		"/send",
		map[string]interface{}{
			"recipient": customerID,
			"channel":   "email",
			"template":  template,
			"data":      data,
		},
		nil,
	)
}

func (w *Worker) postJSON(
	ctx context.Context,
	baseURL string,
	path string,
	payload interface{},
	response interface{},
) error {
	return w.doJSON(
		ctx,
		http.MethodPost,
		baseURL,
		path,
		payload,
		response,
	)
}

func (w *Worker) putJSON(
	ctx context.Context,
	baseURL string,
	path string,
	payload interface{},
	response interface{},
) error {
	return w.doJSON(
		ctx,
		http.MethodPut,
		baseURL,
		path,
		payload,
		response,
	)
}

func (w *Worker) doJSON(
	ctx context.Context,
	method string,
	baseURL string,
	path string,
	payload interface{},
	response interface{},
) error {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	path = "/" + strings.TrimLeft(path, "/")

	if baseURL == "" {
		return fmt.Errorf("service URL is empty")
	}

	requestURL := baseURL + path

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal request body: %w", err)
	}

	requestCtx, cancel := context.WithTimeout(
		ctx,
		w.httpClient.Timeout,
	)
	defer cancel()

	req, err := http.NewRequestWithContext(
		requestCtx,
		method,
		requestURL,
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := w.httpClient.Do(req)
	if err != nil {
		if errors.Is(requestCtx.Err(), context.Canceled) {
			return requestCtx.Err()
		}

		if errors.Is(requestCtx.Err(), context.DeadlineExceeded) {
			return fmt.Errorf("HTTP request timed out: %w", requestCtx.Err())
		}

		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	responseBody, readErr := io.ReadAll(
		io.LimitReader(resp.Body, 1<<20),
	)
	if readErr != nil {
		return fmt.Errorf("read HTTP response: %w", readErr)
	}

	if resp.StatusCode < http.StatusOK ||
		resp.StatusCode >= http.StatusMultipleChoices {
		message := strings.TrimSpace(string(responseBody))

		if message == "" {
			message = resp.Status
		}

		return fmt.Errorf(
			"downstream request failed: %s: %s",
			resp.Status,
			message,
		)
	}

	if response != nil && len(responseBody) > 0 {
		if err := json.Unmarshal(responseBody, response); err != nil {
			return fmt.Errorf(
				"decode downstream response: %w",
				err,
			)
		}
	}

	return nil
}

func requiredString(
	payload map[string]interface{},
	key string,
) (string, error) {
	value, ok := payload[key]

	if !ok {
		return "", fmt.Errorf("%s is required", key)
	}

	stringValue, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("%s must be a string", key)
	}

	stringValue = strings.TrimSpace(stringValue)

	if stringValue == "" {
		return "", fmt.Errorf("%s is required", key)
	}

	return stringValue, nil
}

func stringValue(
	payload map[string]interface{},
	key string,
	fallback string,
) string {
	value, ok := payload[key]
	if !ok {
		return fallback
	}

	stringValue, ok := value.(string)
	if !ok {
		return fallback
	}

	stringValue = strings.TrimSpace(stringValue)

	if stringValue == "" {
		return fallback
	}

	return stringValue
}

func requiredInt(
	payload map[string]interface{},
	key string,
) (int, error) {
	value, ok := payload[key]

	if !ok {
		return 0, fmt.Errorf("%s is required", key)
	}

	switch number := value.(type) {
	case float64:
		if number <= 0 || number != float64(int(number)) {
			return 0, fmt.Errorf("%s must be a positive integer", key)
		}

		return int(number), nil

	case int:
		if number <= 0 {
			return 0, fmt.Errorf("%s must be a positive integer", key)
		}

		return number, nil

	case int64:
		if number <= 0 {
			return 0, fmt.Errorf("%s must be a positive integer", key)
		}

		return int(number), nil

	default:
		return 0, fmt.Errorf("%s must be a number", key)
	}
}

func requiredNumber(
	payload map[string]interface{},
	key string,
) (float64, error) {
	value, ok := payload[key]

	if !ok {
		return 0, fmt.Errorf("%s is required", key)
	}

	number, ok := numberValue(value)
	if !ok || number <= 0 {
		return 0, fmt.Errorf("%s must be greater than zero", key)
	}

	return number, nil
}

func optionalNumber(
	payload map[string]interface{},
	key string,
) (float64, error) {
	value, ok := payload[key]

	if !ok || value == nil {
		return 0, nil
	}

	number, ok := numberValue(value)
	if !ok {
		return 0, fmt.Errorf("%s must be a number", key)
	}

	return number, nil
}

func numberValue(value interface{}) (float64, bool) {
	switch number := value.(type) {
	case float64:
		return number, true
	case float32:
		return float64(number), true
	case int:
		return float64(number), true
	case int64:
		return float64(number), true
	case int32:
		return float64(number), true
	default:
		return 0, false
	}
}

func getEnv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}

	return fallback
}

func getDurationEnv(
	key string,
	fallback time.Duration,
) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))

	if value == "" {
		return fallback
	}

	duration, err := time.ParseDuration(value)
	if err != nil || duration <= 0 {
		log.Printf(
			"invalid %s=%q, using default %s",
			key,
			value,
			fallback,
		)

		return fallback
	}

	return duration
}

func getInt32Env(
	key string,
	fallback int32,
) int32 {
	value := strings.TrimSpace(os.Getenv(key))

	if value == "" {
		return fallback
	}

	var parsed int
	if _, err := fmt.Sscanf(value, "%d", &parsed); err != nil ||
		parsed <= 0 {
		log.Printf(
			"invalid %s=%q, using default %d",
			key,
			value,
			fallback,
		)

		return fallback
	}

	return int32(parsed)
}