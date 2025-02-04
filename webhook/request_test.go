package webhook

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/matryer/is"

	"github.com/hotosm/central-webhook/parser"
)

func TestSendRequest(t *testing.T) {
	is := is.New(t)
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// Set up a mock server
	var receivedPayload parser.ProcessedEvent
	var receivedApiKey string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify content type
		is.Equal("application/json", r.Header.Get("Content-Type"))

		// Verify API key was received
		receivedApiKey = r.Header.Get("X-API-Key")

		// Read and parse request body
		body, err := io.ReadAll(r.Body)
		is.NoErr(err)
		defer r.Body.Close()

		err = json.Unmarshal(body, &receivedPayload)
		is.NoErr(err)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Define test cases
	testCases := []struct {
		name         string
		event        parser.ProcessedEvent
		expectedId   string
		expectedType string
		expectedData interface{}
	}{
		{
			name: "Submission Create Event",
			event: parser.ProcessedEvent{
				ID:   "23dc865a-4757-431e-b182-67e7d5581c81",
				Type: "submission.create",
				Data: "<submission>XML Data</submission>",
			},
			expectedId:   "23dc865a-4757-431e-b182-67e7d5581c81",
			expectedType: "submission.create",
			expectedData: "<submission>XML Data</submission>",
		},
		{
			name: "Entity Update Event",
			event: parser.ProcessedEvent{
				ID:   "45fgh789-e32c-56d2-a765-427654321abc",
				Type: "entity.update.version",
				Data: "{\"field\":\"value\"}",
			},
			expectedId:   "45fgh789-e32c-56d2-a765-427654321abc",
			expectedType: "entity.update.version",
			expectedData: "{\"field\":\"value\"}",
		},
		{
			name: "Submission Review Event",
			event: parser.ProcessedEvent{
				ID:   "45fgh789-e32c-56d2-a765-427654321abc",
				Type: "submission.update",
				Data: "approved",
			},
			expectedId:   "45fgh789-e32c-56d2-a765-427654321abc",
			expectedType: "submission.update",
			expectedData: "approved",
		},
	}

	// Execute test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			testApiKey := "test-api-key"
			SendRequest(log, ctx, server.URL, tc.event, &testApiKey)

			// Validate the received payload
			is.Equal(tc.expectedId, receivedPayload.ID)
			is.Equal(tc.expectedType, receivedPayload.Type)
			is.Equal(tc.expectedData, receivedPayload.Data)

			// Validate that the API key header was sent correctly
			is.Equal("test-api-key", receivedApiKey)
		})
	}
}
