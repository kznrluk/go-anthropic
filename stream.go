package anthropic

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type (
	EventType string
	RoleType  string

	MessageResponse struct {
		// Delta is the response from the stream
		Delta Content

		// Content is the response from the REST API
		Content []Content
	}

	MessageRequest struct {
		MaxTokens int       `json:"max_tokens"`
		Model     ModelType `json:"model"`
		System    string    `json:"system"`
		Messages  []Message `json:"messages"`

		Stream bool `json:"stream"`
	}

	Message struct {
		Role    RoleType `json:"role"`
		Content string   `json:"content"`
	}

	RawResponse struct {
		Type EventType   `json:"type"`
		Data interface{} `json:"data"`
	}

	ContentBlockDelta struct {
		Type  EventType `json:"type"`
		Index int       `json:"index"`
		Delta Content   `json:"delta"`
	}

	Content struct {
		Type EventType `json:"type"`
		Text string    `json:"text"`
	}

	Stream struct {
		reader     *bufio.Reader
		response   *http.Response
		isFinished bool
	}
)

const (
	ContentBlockType EventType = "content_block_delta"

	ChatMessageRoleUser      RoleType = "user"
	ChatMessageRoleAssistant RoleType = "assistant"
)

var (
	dataPrefix = []byte("data: ")
)

func (c *Client) CreateMessage(reqBody MessageRequest) (*MessageResponse, error) {
	reqBody.Stream = false

	reqData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("error marshaling JSON: %w", err)
	}

	url := c.baseUrl + "/messages"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(reqData))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("anthropic-beta", "messages-2023-12-15")
	req.Header.Set("content-type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}

	respStr, err := io.ReadAll(resp.Body)

	var rawResp RawResponse
	if err := json.Unmarshal(respStr, &rawResp); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	if rawResp.Type == "message" {
		var MessageResponse MessageResponse
		if err := json.Unmarshal(respStr, &MessageResponse); err != nil {
			return nil, fmt.Errorf("error decoding response: %w", err)
		}

		return &MessageResponse, nil
	}

	return &MessageResponse{}, nil
}

func (c *Client) CreateMessageStream(reqBody MessageRequest) (*Stream, error) {
	reqBody.Stream = true

	reqData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("error marshaling JSON: %w", err)
	}

	url := c.baseUrl + "/messages"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(reqData))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("anthropic-beta", "messages-2023-12-15")
	req.Header.Set("content-type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}

	return &Stream{
		reader:     bufio.NewReader(resp.Body),
		response:   resp,
		isFinished: false,
	}, nil
}

func (s *Stream) Recv() (MessageResponse, error) {
	if s.isFinished {
		return MessageResponse{}, io.EOF
	}

	for {
		rawLine, err := s.reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				s.isFinished = true
			}
			return MessageResponse{}, err
		}

		if bytes.HasPrefix(rawLine, dataPrefix) {
			rawLine = bytes.TrimPrefix(rawLine, dataPrefix)

			var resp RawResponse
			if err := json.Unmarshal(rawLine, &resp); err != nil {
				return MessageResponse{}, err
			}

			if resp.Type == ContentBlockType {
				var delta ContentBlockDelta
				if err := json.Unmarshal(rawLine, &delta); err != nil {
					return MessageResponse{}, err
				}

				return MessageResponse{
					Delta: delta.Delta,
				}, nil
			}

			continue
		}
	}
}

func (s *Stream) Close() error {
	return s.response.Body.Close()
}
