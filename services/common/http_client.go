package common

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

func NewHTTPClient() *http.Client {
	return &http.Client{
		Timeout:   5 * time.Second,
		Transport: otelTransport,
	}
}

func PostJSON(ctx context.Context, client *http.Client, url string, requestBody any, responseBody any) error {
	log.Printf("outbound method=POST url=%s", url)
	payload, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("marshal request body: %w", err)
	}

	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	httpRequest.Header.Set("Content-Type", "application/json")

	httpResponse, err := client.Do(httpRequest)
	if err != nil {
		log.Printf("outbound method=POST url=%s error=%v", url, err)
		return fmt.Errorf("post %s: %w", url, err)
	}
	defer httpResponse.Body.Close()

	body, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if httpResponse.StatusCode >= http.StatusBadRequest {
		log.Printf("outbound method=POST url=%s status=%s", url, httpResponse.Status)
		return fmt.Errorf("request failed with status %s: %s", httpResponse.Status, string(body))
	}

	log.Printf("outbound method=POST url=%s status=%s", url, httpResponse.Status)

	if responseBody == nil {
		return nil
	}

	if err := json.Unmarshal(body, responseBody); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	return nil
}
