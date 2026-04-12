package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeRejectsUnknownFields(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"TaskFlow","extra":"nope"}`))
	var payload struct {
		Name string `json:"name"`
	}

	err := decode(request, &payload)
	if err == nil {
		t.Fatal("expected decode to reject unknown field")
	}
}

func TestEncodeWritesJSONResponse(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()

	encode(recorder, http.StatusCreated, map[string]string{"status": "ok"})

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, recorder.Code)
	}

	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected content type application/json, got %q", got)
	}

	var body map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected valid json body: %v", err)
	}

	if body["status"] != "ok" {
		t.Fatalf("expected status ok, got %q", body["status"])
	}
}

func TestWriteValidationErrorShape(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()

	writeValidationError(recorder, map[string]string{"email": "is required"})

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}

	var body struct {
		Error  string            `json:"error"`
		Fields map[string]string `json:"fields"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected valid json body: %v", err)
	}

	if body.Error != "validation failed" {
		t.Fatalf("expected validation failed, got %q", body.Error)
	}

	if body.Fields["email"] != "is required" {
		t.Fatalf("expected email validation message, got %q", body.Fields["email"])
	}
}
