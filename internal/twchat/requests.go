package twchat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// All request types implement twapi.HTTPRequester. The server argument is the
// Teamwork installation base URL (e.g. https://acme.teamwork.com); the engine's
// session fills in the host and Authorization header when it is left empty. We
// only build the path-relative URL under /chat/v7/.

const chatBasePath = "/chat/v7"

// currentUserGetRequest fetches the current authenticated chat user.
type currentUserGetRequest struct{}

// HTTPRequest builds the GET /chat/v7/me request.
func (currentUserGetRequest) HTTPRequest(ctx context.Context, server string) (*http.Request, error) {
	return http.NewRequestWithContext(ctx, http.MethodGet, server+chatBasePath+"/me", nil)
}

// conversationListRequest lists conversations for the current user.
type conversationListRequest struct {
	PageOffset         int
	PageLimit          int
	SearchTerm         string
	Status             string
	Type               string
	Sort               string
	IncludeMessageData bool
}

// HTTPRequest builds the GET /chat/v7/conversations request.
func (c conversationListRequest) HTTPRequest(ctx context.Context, server string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server+chatBasePath+"/conversations", nil)
	if err != nil {
		return nil, err
	}
	q := url.Values{}
	if c.PageOffset > 0 {
		q.Set("page[offset]", strconv.Itoa(c.PageOffset))
	}
	if c.PageLimit > 0 {
		q.Set("page[limit]", strconv.Itoa(c.PageLimit))
	}
	if c.SearchTerm != "" {
		q.Set("filter[searchTerm]", c.SearchTerm)
	}
	if c.Status != "" {
		q.Set("filter[status]", c.Status)
	}
	if c.Type != "" {
		q.Set("filter[type]", c.Type)
	}
	if c.Sort != "" {
		q.Set("sort", c.Sort)
	}
	if c.IncludeMessageData {
		q.Set("includeMessageData", "true")
	}
	req.URL.RawQuery = q.Encode()
	return req, nil
}

// conversationGetRequest fetches a single conversation by ID.
type conversationGetRequest struct {
	ID int64
}

// HTTPRequest builds the GET /chat/v7/conversations/{id} request.
func (c conversationGetRequest) HTTPRequest(ctx context.Context, server string) (*http.Request, error) {
	uri := server + chatBasePath + "/conversations/" + strconv.FormatInt(c.ID, 10)
	return http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
}

// pairConversationGetRequest gets (or creates) the 1:1 "pair" conversation
// between the current user and the given person.
type pairConversationGetRequest struct {
	UserID int64
}

// HTTPRequest builds the GET /chat/v7/people/{id}/conversation request.
func (p pairConversationGetRequest) HTTPRequest(ctx context.Context, server string) (*http.Request, error) {
	uri := server + chatBasePath + "/people/" + strconv.FormatInt(p.UserID, 10) + "/conversation"
	return http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
}

// messageListRequest lists messages within a conversation.
type messageListRequest struct {
	ConversationID  int64
	Page            int
	PageSize        int
	SearchTerm      string
	BeforeMessageID int64
	AfterMessageID  int64
	CreatedBefore   string
	CreatedAfter    string
}

// HTTPRequest builds the GET /chat/v7/conversations/{id}/messages request.
func (m messageListRequest) HTTPRequest(ctx context.Context, server string) (*http.Request, error) {
	uri := server + chatBasePath + "/conversations/" + strconv.FormatInt(m.ConversationID, 10) + "/messages"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}
	q := url.Values{}
	if m.Page > 0 {
		q.Set("page", strconv.Itoa(m.Page))
	}
	if m.PageSize > 0 {
		q.Set("pageSize", strconv.Itoa(m.PageSize))
	}
	if m.SearchTerm != "" {
		q.Set("filter[searchTerm]", m.SearchTerm)
	}
	if m.BeforeMessageID > 0 {
		q.Set("beforeMessageId", strconv.FormatInt(m.BeforeMessageID, 10))
	}
	if m.AfterMessageID > 0 {
		q.Set("afterMessageId", strconv.FormatInt(m.AfterMessageID, 10))
	}
	if m.CreatedBefore != "" {
		q.Set("createdBefore", m.CreatedBefore)
	}
	if m.CreatedAfter != "" {
		q.Set("createdAfter", m.CreatedAfter)
	}
	req.URL.RawQuery = q.Encode()
	return req, nil
}

// peopleListRequest lists people in the installation.
type peopleListRequest struct {
	PageOffset int
	PageLimit  int
	SearchTerm string
}

// HTTPRequest builds the GET /chat/v7/people request.
func (p peopleListRequest) HTTPRequest(ctx context.Context, server string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server+chatBasePath+"/people", nil)
	if err != nil {
		return nil, err
	}
	q := url.Values{}
	if p.PageOffset > 0 {
		q.Set("page[offset]", strconv.Itoa(p.PageOffset))
	}
	if p.PageLimit > 0 {
		q.Set("page[limit]", strconv.Itoa(p.PageLimit))
	}
	if p.SearchTerm != "" {
		q.Set("filter[searchTerm]", p.SearchTerm)
	}
	req.URL.RawQuery = q.Encode()
	return req, nil
}

// messageSendRequest posts a message to a conversation.
type messageSendRequest struct {
	ConversationID int64
	Body           string
}

// HTTPRequest builds the POST /chat/v7/conversations/{id}/messages request.
func (m messageSendRequest) HTTPRequest(ctx context.Context, server string) (*http.Request, error) {
	uri := server + chatBasePath + "/conversations/" + strconv.FormatInt(m.ConversationID, 10) + "/messages"

	payload := struct {
		ConversationID int64 `json:"conversationId"`
		Message        struct {
			Body string `json:"body"`
		} `json:"message"`
	}{ConversationID: m.ConversationID}
	payload.Message.Body = m.Body

	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(payload); err != nil {
		return nil, fmt.Errorf("failed to encode send message request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uri, &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}
