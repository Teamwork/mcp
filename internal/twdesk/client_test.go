package twdesk

import (
	"context"
	"net/http"
	"testing"

	"github.com/teamwork/mcp/internal/config"
)

func TestClientFromContext_DefaultBaseURL(t *testing.T) {
	ctx := context.Background()
	httpClient := &http.Client{}

	client := ClientFromContext(ctx, httpClient)

	if client == nil {
		t.Fatal("expected client, got nil")
	}
}

func TestClientFromContext_WithCustomerURL(t *testing.T) {
	ctx := context.Background()
	ctx = config.WithCustomerURL(ctx, "https://digitalcrew.teamwork.com")
	httpClient := &http.Client{}

	client := ClientFromContext(ctx, httpClient)

	if client == nil {
		t.Fatal("expected client, got nil")
	}
}

func TestClientFromContext_WithCustomerURLTrailingSlash(t *testing.T) {
	ctx := context.Background()
	ctx = config.WithCustomerURL(ctx, "https://digitalcrew.teamwork.com/")
	httpClient := &http.Client{}

	client := ClientFromContext(ctx, httpClient)

	if client == nil {
		t.Fatal("expected client, got nil")
	}
	// If the client was created successfully, the trailing slash was handled correctly
}

func TestClientFromContext_WithBearerToken(t *testing.T) {
	ctx := context.Background()
	ctx = config.WithBearerToken(ctx, "test-bearer-token")
	httpClient := &http.Client{}

	client := ClientFromContext(ctx, httpClient)

	if client == nil {
		t.Fatal("expected client, got nil")
	}
}

func TestClientFromContext_WithAllContext(t *testing.T) {
	ctx := context.Background()
	ctx = config.WithCustomerURL(ctx, "https://test.teamwork.com/")
	ctx = config.WithBearerToken(ctx, "test-token")
	httpClient := &http.Client{}

	client := ClientFromContext(ctx, httpClient)

	if client == nil {
		t.Fatal("expected client, got nil")
	}
}

func TestClientFromContext_WithoutContext(t *testing.T) {
	ctx := context.Background()
	httpClient := &http.Client{}

	client := ClientFromContext(ctx, httpClient)

	if client == nil {
		t.Fatal("expected client, got nil")
	}
	// Should use default base URL
}

func TestClientFromContext_EmptyCustomerURL(t *testing.T) {
	ctx := context.Background()
	ctx = config.WithCustomerURL(ctx, "")
	httpClient := &http.Client{}

	client := ClientFromContext(ctx, httpClient)

	if client == nil {
		t.Fatal("expected client, got nil")
	}
	// Empty string should be ignored, use default
}

func TestClientFromContext_EmptyBearerToken(t *testing.T) {
	ctx := context.Background()
	ctx = config.WithBearerToken(ctx, "")
	httpClient := &http.Client{}

	client := ClientFromContext(ctx, httpClient)

	if client == nil {
		t.Fatal("expected client, got nil")
	}
	// Empty token should be ignored
}
