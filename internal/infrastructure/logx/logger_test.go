package logx

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"
)

const (
	WorkerUUID  string = "worker-123"
	RefreshUUID string = "refresh-123"
	RequestUUID string = "request-123"
)

// TestWithContextAllKeys тестирует, что все ключи,
// которые WithContext умеет идентифицировать в ctx,
// добавляются в объект логгера.
func TestWithContextAllKeys(t *testing.T) {
	var buf bytes.Buffer

	logger := initLogger(&buf)
	ctx := initCtx(map[string]string{
		WorkerID:  WorkerUUID,
		RefreshID: RefreshUUID,
		HttpID:    RequestUUID,
	})

	extLogger := WithContext(ctx, logger)
	extLogger.Info("test message")

	logRecord := parseLogRecord(t, &buf)

	assertFieldValue(t, logRecord, WorkerID, WorkerUUID)
	assertFieldValue(t, logRecord, RefreshID, RefreshUUID)
	assertFieldValue(t, logRecord, HttpID, RequestUUID)
}

// TestWithContextOnlyRequest тестирует,
// что в логгер добавляется только request_id.
func TestWithContextOnlyRequest(t *testing.T) {
	var buf bytes.Buffer

	logger := initLogger(&buf)
	ctx := initCtx(map[string]string{
		HttpID: RequestUUID,
	})

	extLogger := WithContext(ctx, logger)
	extLogger.Info("test message")

	logRecord := parseLogRecord(t, &buf)

	assertFieldAbsent(t, logRecord, WorkerID)
	assertFieldAbsent(t, logRecord, RefreshID)
	assertFieldValue(t, logRecord, HttpID, RequestUUID)
}

// TestWithContextOnlyWorker тестирует,
// что в логгер добавляется только worker_id.
func TestWithContextOnlyWorker(t *testing.T) {
	var buf bytes.Buffer

	logger := initLogger(&buf)
	ctx := initCtx(map[string]string{
		WorkerID: WorkerUUID,
	})

	extLogger := WithContext(ctx, logger)
	extLogger.Info("test message")

	logRecord := parseLogRecord(t, &buf)

	assertFieldValue(t, logRecord, WorkerID, WorkerUUID)
	assertFieldAbsent(t, logRecord, RefreshID)
	assertFieldAbsent(t, logRecord, HttpID)
}

// TestWithContextOnlyRefresh тестирует,
// что в логгер добавляется только refresh_id.
func TestWithContextOnlyRefresh(t *testing.T) {
	var buf bytes.Buffer

	logger := initLogger(&buf)
	ctx := initCtx(map[string]string{
		RefreshID: RefreshUUID,
	})

	extLogger := WithContext(ctx, logger)
	extLogger.Info("test message")

	logRecord := parseLogRecord(t, &buf)

	assertFieldAbsent(t, logRecord, WorkerID)
	assertFieldValue(t, logRecord, RefreshID, RefreshUUID)
	assertFieldAbsent(t, logRecord, HttpID)
}

// TestWithContextOtherKeys тестирует,
// что если переданы ключи, которые функция не умеет обрабатывать,
// то в логах они не появляются.
func TestWithContextOtherKeys(t *testing.T) {
	var buf bytes.Buffer

	logger := initLogger(&buf)
	keyUUID := "something-123"

	ctx := initCtx(map[string]string{
		"Something": keyUUID,
	})

	extLogger := WithContext(ctx, logger)
	extLogger.Info("test message")

	logRecord := parseLogRecord(t, &buf)

	assertFieldAbsent(t, logRecord, WorkerID)
	assertFieldAbsent(t, logRecord, RefreshID)
	assertFieldAbsent(t, logRecord, HttpID)

	if got, ok := logRecord["Something"]; ok {
		t.Fatalf("did not expect field %q in log, got %v", "Something", got)
	}
}

// initLogger инициализирует объект логгера.
func initLogger(out *bytes.Buffer) *slog.Logger {
	return slog.New(slog.NewJSONHandler(out, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
}

// initCtx инициализирует context.Context с переданными ключами.
func initCtx(data map[string]string) context.Context {
	ctx := context.Background()
	for k, v := range data {
		ctx = context.WithValue(ctx, k, v)
	}
	return ctx
}

// parseLogRecord парсит JSON-лог из буфера в map.
func parseLogRecord(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()

	var logRecord map[string]any
	if err := json.Unmarshal(buf.Bytes(), &logRecord); err != nil {
		t.Fatalf("failed to unmarshal log output: %v; output: %s", err, buf.String())
	}

	return logRecord
}

// assertFieldValue проверяет, что поле присутствует в логе
// и имеет ожидаемое значение.
func assertFieldValue(t *testing.T, logRecord map[string]any, field string, want string) {
	t.Helper()

	got, ok := logRecord[field]
	if !ok {
		t.Fatalf("expected field %q in log, but it is absent", field)
	}

	gotStr, ok := got.(string)
	if !ok {
		t.Fatalf("expected field %q to be string, got %T", field, got)
	}

	if gotStr != want {
		t.Fatalf("expected %s=%q, got %q", field, want, gotStr)
	}
}

// assertFieldAbsent проверяет, что поле отсутствует в логе.
func assertFieldAbsent(t *testing.T, logRecord map[string]any, field string) {
	t.Helper()

	if got, ok := logRecord[field]; ok {
		t.Fatalf("did not expect field %q in log, got %v", field, got)
	}
}
