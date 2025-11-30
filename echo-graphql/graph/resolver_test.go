package graph_test

import (
	"context"
	"testing"
	"time"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"

	"github.com/jsr-probitas/dockerfiles/echo-graphql/graph"
)

func setupTestClient(t *testing.T) *client.Client {
	t.Helper()
	resolver := graph.NewResolver()
	srv := handler.New(graph.NewExecutableSchema(graph.Config{
		Resolvers: resolver,
	}))
	srv.AddTransport(transport.POST{})
	return client.New(srv)
}

func setupTestResolver(t *testing.T) *graph.Resolver {
	t.Helper()
	return graph.NewResolver()
}

// Query Tests

func TestEcho_ReturnsSameMessage(t *testing.T) {
	c := setupTestClient(t)

	var resp struct {
		Echo string
	}
	c.MustPost(`query { echo(message: "hello") }`, &resp)

	if resp.Echo != "hello" {
		t.Errorf("expected 'hello', got %q", resp.Echo)
	}
}

func TestEchoWithDelay_ReturnsAfterDelay(t *testing.T) {
	c := setupTestClient(t)

	start := time.Now()
	var resp struct {
		EchoWithDelay string
	}
	c.MustPost(`query { echoWithDelay(message: "delayed", delayMs: 10) }`, &resp)
	elapsed := time.Since(start)

	if resp.EchoWithDelay != "delayed" {
		t.Errorf("expected 'delayed', got %q", resp.EchoWithDelay)
	}
	if elapsed < 10*time.Millisecond {
		t.Errorf("expected delay of at least 10ms, got %v", elapsed)
	}
}

func TestEchoError_ReturnsGraphQLError(t *testing.T) {
	c := setupTestClient(t)

	var resp struct {
		EchoError string
	}
	err := c.Post(`query { echoError(message: "test error message") }`, &resp)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if resp.EchoError != "" {
		t.Errorf("expected empty response, got %q", resp.EchoError)
	}
}

func TestEchoPartialError_ReturnsMixedResults(t *testing.T) {
	c := setupTestClient(t)

	var resp struct {
		EchoPartialError []struct {
			Message *string
			Error   *string
		}
	}
	c.MustPost(`query { echoPartialError(messages: ["hello", "error", "world"]) { message error } }`, &resp)

	if len(resp.EchoPartialError) != 3 {
		t.Fatalf("expected 3 results, got %d", len(resp.EchoPartialError))
	}

	// First result: "hello" - should have message, no error
	if resp.EchoPartialError[0].Message == nil || *resp.EchoPartialError[0].Message != "hello" {
		t.Errorf("expected first message to be 'hello', got %v", resp.EchoPartialError[0].Message)
	}
	if resp.EchoPartialError[0].Error != nil {
		t.Errorf("expected first error to be nil, got %v", resp.EchoPartialError[0].Error)
	}

	// Second result: "error" - should have error, no message
	if resp.EchoPartialError[1].Message != nil {
		t.Errorf("expected second message to be nil, got %v", resp.EchoPartialError[1].Message)
	}
	if resp.EchoPartialError[1].Error == nil {
		t.Error("expected second error to be non-nil")
	} else if *resp.EchoPartialError[1].Error != "message contains 'error'" {
		t.Errorf("expected error message 'message contains 'error'', got %q", *resp.EchoPartialError[1].Error)
	}

	// Third result: "world" - should have message, no error
	if resp.EchoPartialError[2].Message == nil || *resp.EchoPartialError[2].Message != "world" {
		t.Errorf("expected third message to be 'world', got %v", resp.EchoPartialError[2].Message)
	}
	if resp.EchoPartialError[2].Error != nil {
		t.Errorf("expected third error to be nil, got %v", resp.EchoPartialError[2].Error)
	}
}

func TestEchoPartialError_ContainsErrorSubstring(t *testing.T) {
	c := setupTestClient(t)

	testCases := []struct {
		input       string
		shouldError bool
	}{
		{"success", false},
		{"error", true},
		{"ERROR", true},
		{"this is an error message", true},
		{"errorHandling", true},
		{"no problem here", false},
		{"Error: something went wrong", true},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			var resp struct {
				EchoPartialError []struct {
					Message *string
					Error   *string
				}
			}
			query := `query { echoPartialError(messages: ["` + tc.input + `"]) { message error } }`
			c.MustPost(query, &resp)

			if len(resp.EchoPartialError) != 1 {
				t.Fatalf("expected 1 result, got %d", len(resp.EchoPartialError))
			}

			result := resp.EchoPartialError[0]
			if tc.shouldError {
				if result.Message != nil {
					t.Errorf("expected message to be nil for %q, got %v", tc.input, *result.Message)
				}
				if result.Error == nil {
					t.Errorf("expected error to be non-nil for %q", tc.input)
				} else if *result.Error != "message contains 'error'" {
					t.Errorf("expected error message 'message contains 'error'', got %q", *result.Error)
				}
			} else {
				if result.Message == nil || *result.Message != tc.input {
					t.Errorf("expected message to be %q, got %v", tc.input, result.Message)
				}
				if result.Error != nil {
					t.Errorf("expected error to be nil for %q, got %v", tc.input, *result.Error)
				}
			}
		})
	}
}

func TestEchoWithExtensions_ReturnsMessage(t *testing.T) {
	c := setupTestClient(t)

	var resp struct {
		EchoWithExtensions string
	}
	c.MustPost(`query { echoWithExtensions(message: "with extensions") }`, &resp)

	if resp.EchoWithExtensions != "with extensions" {
		t.Errorf("expected 'with extensions', got %q", resp.EchoWithExtensions)
	}
}

// Mutation Tests

func TestCreateMessage_CreatesAndReturnsMessage(t *testing.T) {
	c := setupTestClient(t)

	var resp struct {
		CreateMessage struct {
			ID        string
			Text      string
			CreatedAt string
		}
	}
	c.MustPost(`mutation { createMessage(text: "hello world") { id text createdAt } }`, &resp)

	if resp.CreateMessage.ID == "" {
		t.Error("expected non-empty ID")
	}
	if resp.CreateMessage.Text != "hello world" {
		t.Errorf("expected text 'hello world', got %q", resp.CreateMessage.Text)
	}
	if resp.CreateMessage.CreatedAt == "" {
		t.Error("expected non-empty createdAt")
	}
}

func TestUpdateMessage_UpdatesExistingMessage(t *testing.T) {
	c := setupTestClient(t)

	// Create a message first
	var createResp struct {
		CreateMessage struct {
			ID   string
			Text string
		}
	}
	c.MustPost(`mutation { createMessage(text: "original") { id text } }`, &createResp)

	// Update the message
	var updateResp struct {
		UpdateMessage struct {
			ID   string
			Text string
		}
	}
	query := `mutation { updateMessage(id: "` + createResp.CreateMessage.ID + `", text: "updated") { id text } }`
	c.MustPost(query, &updateResp)

	if updateResp.UpdateMessage.ID != createResp.CreateMessage.ID {
		t.Errorf("expected ID %q, got %q", createResp.CreateMessage.ID, updateResp.UpdateMessage.ID)
	}
	if updateResp.UpdateMessage.Text != "updated" {
		t.Errorf("expected text 'updated', got %q", updateResp.UpdateMessage.Text)
	}
}

func TestUpdateMessage_ReturnsErrorForNonExistentID(t *testing.T) {
	c := setupTestClient(t)

	var resp struct {
		UpdateMessage *struct {
			ID   string
			Text string
		}
	}
	err := c.Post(`mutation { updateMessage(id: "non-existent", text: "updated") { id text } }`, &resp)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDeleteMessage_ReturnsTrueForExistingMessage(t *testing.T) {
	c := setupTestClient(t)

	// Create a message first
	var createResp struct {
		CreateMessage struct {
			ID string
		}
	}
	c.MustPost(`mutation { createMessage(text: "to delete") { id } }`, &createResp)

	// Delete the message
	var deleteResp struct {
		DeleteMessage bool
	}
	query := `mutation { deleteMessage(id: "` + createResp.CreateMessage.ID + `") }`
	c.MustPost(query, &deleteResp)

	if !deleteResp.DeleteMessage {
		t.Error("expected deleteMessage to return true for existing message")
	}
}

func TestDeleteMessage_ReturnsFalseForNonExistentID(t *testing.T) {
	c := setupTestClient(t)

	var resp struct {
		DeleteMessage bool
	}
	c.MustPost(`mutation { deleteMessage(id: "non-existent") }`, &resp)

	if resp.DeleteMessage {
		t.Error("expected deleteMessage to return false for non-existent ID")
	}
}

// Subscription Tests

func TestCountdown_EmitsCorrectSequence(t *testing.T) {
	resolver := setupTestResolver(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	subResolver := resolver.Subscription()
	ch, err := subResolver.Countdown(ctx, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []int{3, 2, 1, 0}
	received := []int{}

	for val := range ch {
		received = append(received, val)
		if len(received) >= len(expected) {
			break
		}
	}

	if len(received) != len(expected) {
		t.Fatalf("expected %d values, got %d", len(expected), len(received))
	}

	for i, exp := range expected {
		if received[i] != exp {
			t.Errorf("at index %d: expected %d, got %d", i, exp, received[i])
		}
	}
}
