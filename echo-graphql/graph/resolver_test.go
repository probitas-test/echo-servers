package graph_test

import (
	"context"
	"testing"
	"time"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"

	"github.com/jsr-probitas/echo-servers/echo-graphql/graph"
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

func TestEchoNested_ReturnsNestedStructure(t *testing.T) {
	c := setupTestClient(t)

	var resp struct {
		EchoNested struct {
			Value string
			Child *struct {
				Value string
				Child *struct {
					Value string
					Child *struct {
						Value string
						Child *struct {
							Value string
						}
					}
				}
			}
		}
	}
	c.MustPost(`query { echoNested(message: "test", depth: 3) { value child { value child { value child { value } } } } }`, &resp)

	if resp.EchoNested.Value != "test (level 1)" {
		t.Errorf("expected 'test (level 1)', got %q", resp.EchoNested.Value)
	}
	if resp.EchoNested.Child == nil {
		t.Fatal("expected child at level 2")
	}
	if resp.EchoNested.Child.Value != "test (level 2)" {
		t.Errorf("expected 'test (level 2)', got %q", resp.EchoNested.Child.Value)
	}
	if resp.EchoNested.Child.Child == nil {
		t.Fatal("expected child at level 3")
	}
	if resp.EchoNested.Child.Child.Value != "test (level 3)" {
		t.Errorf("expected 'test (level 3)', got %q", resp.EchoNested.Child.Child.Value)
	}
	if resp.EchoNested.Child.Child.Child != nil {
		t.Error("expected no child beyond depth 3")
	}
}

func TestEchoNested_DepthOne(t *testing.T) {
	c := setupTestClient(t)

	var resp struct {
		EchoNested struct {
			Value string
			Child *struct {
				Value string
			}
		}
	}
	c.MustPost(`query { echoNested(message: "single", depth: 1) { value child { value } } }`, &resp)

	if resp.EchoNested.Value != "single (level 1)" {
		t.Errorf("expected 'single (level 1)', got %q", resp.EchoNested.Value)
	}
	if resp.EchoNested.Child != nil {
		t.Error("expected no child for depth 1")
	}
}

func TestEchoList_ReturnsCorrectCount(t *testing.T) {
	c := setupTestClient(t)

	var resp struct {
		EchoList []struct {
			Index   int
			Message string
		}
	}
	c.MustPost(`query { echoList(message: "item", count: 5) { index message } }`, &resp)

	if len(resp.EchoList) != 5 {
		t.Fatalf("expected 5 items, got %d", len(resp.EchoList))
	}

	for i, item := range resp.EchoList {
		if item.Index != i {
			t.Errorf("expected index %d, got %d", i, item.Index)
		}
		if item.Message != "item" {
			t.Errorf("expected message 'item', got %q", item.Message)
		}
	}
}

func TestEchoList_EmptyList(t *testing.T) {
	c := setupTestClient(t)

	var resp struct {
		EchoList []struct {
			Index   int
			Message string
		}
	}
	c.MustPost(`query { echoList(message: "item", count: 0) { index message } }`, &resp)

	if len(resp.EchoList) != 0 {
		t.Errorf("expected 0 items, got %d", len(resp.EchoList))
	}
}

func TestEchoNull_ReturnsNull(t *testing.T) {
	c := setupTestClient(t)

	var resp struct {
		EchoNull *string
	}
	c.MustPost(`query { echoNull }`, &resp)

	if resp.EchoNull != nil {
		t.Errorf("expected null, got %q", *resp.EchoNull)
	}
}

func TestEchoOptional_ReturnsValueWhenNotNull(t *testing.T) {
	c := setupTestClient(t)

	var resp struct {
		EchoOptional *string
	}
	c.MustPost(`query { echoOptional(message: "hello", returnNull: false) }`, &resp)

	if resp.EchoOptional == nil {
		t.Fatal("expected non-null value")
	}
	if *resp.EchoOptional != "hello" {
		t.Errorf("expected 'hello', got %q", *resp.EchoOptional)
	}
}

func TestEchoOptional_ReturnsNullWhenRequested(t *testing.T) {
	c := setupTestClient(t)

	var resp struct {
		EchoOptional *string
	}
	c.MustPost(`query { echoOptional(message: "hello", returnNull: true) }`, &resp)

	if resp.EchoOptional != nil {
		t.Errorf("expected null, got %q", *resp.EchoOptional)
	}
}

func TestEchoHeaders_ReturnsEmptyWhenNoRequest(t *testing.T) {
	c := setupTestClient(t)

	var resp struct {
		EchoHeaders struct {
			Authorization *string
			ContentType   *string
			All           []struct {
				Name  string
				Value string
			}
		}
	}
	c.MustPost(`query { echoHeaders { authorization contentType all { name value } } }`, &resp)

	// Without the middleware, request is nil, so headers are empty
	if resp.EchoHeaders.Authorization != nil {
		t.Errorf("expected authorization to be nil without request context")
	}
	if len(resp.EchoHeaders.All) != 0 {
		t.Errorf("expected empty all headers without request context")
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

func TestBatchCreateMessages_CreatesMultipleMessages(t *testing.T) {
	c := setupTestClient(t)

	var resp struct {
		BatchCreateMessages []struct {
			ID        string
			Text      string
			CreatedAt string
		}
	}
	c.MustPost(`mutation { batchCreateMessages(texts: ["first", "second", "third"]) { id text createdAt } }`, &resp)

	if len(resp.BatchCreateMessages) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(resp.BatchCreateMessages))
	}

	expectedTexts := []string{"first", "second", "third"}
	for i, msg := range resp.BatchCreateMessages {
		if msg.ID == "" {
			t.Errorf("expected non-empty ID at index %d", i)
		}
		if msg.Text != expectedTexts[i] {
			t.Errorf("expected text %q at index %d, got %q", expectedTexts[i], i, msg.Text)
		}
		if msg.CreatedAt == "" {
			t.Errorf("expected non-empty createdAt at index %d", i)
		}
	}
}

func TestBatchCreateMessages_EmptyList(t *testing.T) {
	c := setupTestClient(t)

	var resp struct {
		BatchCreateMessages []struct {
			ID   string
			Text string
		}
	}
	c.MustPost(`mutation { batchCreateMessages(texts: []) { id text } }`, &resp)

	if len(resp.BatchCreateMessages) != 0 {
		t.Errorf("expected 0 messages, got %d", len(resp.BatchCreateMessages))
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

func TestHeartbeat_EmitsTimestamps(t *testing.T) {
	resolver := setupTestResolver(t)
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	subResolver := resolver.Subscription()
	ch, err := subResolver.Heartbeat(ctx, 50) // 50ms interval
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	count := 0
	for range ch {
		count++
		if count >= 3 {
			break
		}
	}

	if count < 3 {
		t.Errorf("expected at least 3 heartbeats, got %d", count)
	}
}

func TestMessageCreatedFiltered_ReceivesMatchingMessages(t *testing.T) {
	resolver := setupTestResolver(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	filter := "important"
	subResolver := resolver.Subscription()
	ch, err := subResolver.MessageCreatedFiltered(ctx, &filter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Create messages in a goroutine
	go func() {
		time.Sleep(10 * time.Millisecond)
		mutResolver := resolver.Mutation()
		_, _ = mutResolver.CreateMessage(ctx, "not matching")
		_, _ = mutResolver.CreateMessage(ctx, "important message")
		_, _ = mutResolver.CreateMessage(ctx, "another not matching")
		_, _ = mutResolver.CreateMessage(ctx, "very important")
	}()

	received := []*string{}
	timeout := time.After(500 * time.Millisecond)
	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				goto done
			}
			received = append(received, &msg.Text)
			if len(received) >= 2 {
				goto done
			}
		case <-timeout:
			goto done
		}
	}
done:

	if len(received) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(received))
	}
	if *received[0] != "important message" {
		t.Errorf("expected 'important message', got %q", *received[0])
	}
	if *received[1] != "very important" {
		t.Errorf("expected 'very important', got %q", *received[1])
	}
}

func TestMessageCreatedFiltered_NoFilter_ReceivesAll(t *testing.T) {
	resolver := setupTestResolver(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	subResolver := resolver.Subscription()
	ch, err := subResolver.MessageCreatedFiltered(ctx, nil) // nil filter = receive all
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Create messages
	go func() {
		time.Sleep(10 * time.Millisecond)
		mutResolver := resolver.Mutation()
		_, _ = mutResolver.CreateMessage(ctx, "first")
		_, _ = mutResolver.CreateMessage(ctx, "second")
	}()

	received := []string{}
	timeout := time.After(500 * time.Millisecond)
	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				goto done
			}
			received = append(received, msg.Text)
			if len(received) >= 2 {
				goto done
			}
		case <-timeout:
			goto done
		}
	}
done:

	if len(received) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(received))
	}
}
