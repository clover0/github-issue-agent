package models

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/clover0/issue-agent/logger"
	"github.com/clover0/issue-agent/util"
)

type AnthropicClient struct {
	client  *http.Client
	logger  logger.Logger
	baseURL *url.URL

	// services
	Messages *AnthropicMessageService
}

func NewAnthropic(logger logger.Logger, apiKey string) AnthropicClient {
	baseURL, err := url.Parse("https://api.anthropic.com")
	if err != nil {
		logger.Error("failed to parse base URL: %s", err)
	}

	client := &http.Client{
		Transport: roundTripper(func(req *http.Request) (*http.Response, error) {
			req.Header.Set("content-type", "application/json")
			req.Header.Set("x-api-key", apiKey)
			req.Header.Set("anthropic-version", "2023-06-01")
			return http.DefaultTransport.RoundTrip(req)
		}),
	}
	c := AnthropicClient{
		logger:  logger,
		client:  client,
		baseURL: baseURL,
	}

	c.Messages = &AnthropicMessageService{client: &c}

	return c
}

type roundTripper func(r *http.Request) (*http.Response, error)

func (r roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return r(req)
}

func (c *AnthropicClient) NewRequest(method string, path string, body interface{}) (*http.Request, error) {
	u, err := c.baseURL.Parse(path)
	if err != nil {

	}

	b, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal body: %w", err)
	}

	req, err := http.NewRequest(method, u.String(), bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	return req, nil
}

type AnthropicMessageService struct {
	client *AnthropicClient
}

func (s *AnthropicMessageService) Create(ctx context.Context, body J) (*ResponseMessage, error) {
	var message *ResponseMessage
	err := util.Retry(3, func() error {
		req, err := s.client.NewRequest("POST", "v1/messages", body)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		resp, err := s.client.client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to send request: %w", err)
		}
		defer resp.Body.Close()

		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body: %w", err)
		}

		if resp.StatusCode >= 400 {
			s := string(b)
			if resp.StatusCode == http.StatusServiceUnavailable || strings.Contains(s, "overloaded") {
				return util.RetryableError
			}
			return fmt.Errorf("invalid request or server error %s", b)
		}

		if err := json.Unmarshal(b, &message); err != nil {
			return fmt.Errorf("failed to unmarshal response body: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return message, nil
}

type ResponseMessage struct {
	ID           string           `json:"id"`
	Type         string           `json:"type"`
	Role         string           `json:"role"`
	Content      []MessageContent `json:"content"`
	Model        string           `json:"model"`
	StopReason   string           `json:"stop_reason"`
	StopSequence string           `json:"stop_sequence"`
	Usage        struct {
		InputTokens              int `json:"input_tokens"`
		CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
		CacheReadInputTokens     int `json:"cache_read_input_tokens"`
		OutputTokens             int `json:"output_tokens"`
	} `json:"usage"`
}

type MessageContent struct {
	// Text content
	Type string `json:"type"`
	Text string `json:"text"`

	// Tool Use content
	ID    string `json:"id"`
	Name  string `json:"name"`
	Input J      `json:"input"`
}
