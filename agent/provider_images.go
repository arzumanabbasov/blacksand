package agent

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/Autumn-27/norma/llm"
	"github.com/google/uuid"
)

const imageURLBlockType llm.BlockType = "image_url"

// ImageURLBlock represents a native image without changing the pinned SDK.
// URLs are passed to the model provider; the host never downloads the image.
func ImageURLBlock(value string) llm.ContentBlock {
	return llm.ContentBlock{Type: imageURLBlockType, Text: value}
}

func SupportsImageInput(provider llm.Provider) bool {
	p, ok := provider.(interface{ SupportsImageInput() bool })
	return ok && p.SupportsImageInput()
}

func RequestHasImages(req llm.CompletionRequest) bool {
	for _, message := range req.Messages {
		for _, block := range message.Content {
			if block.Type == imageURLBlockType {
				return true
			}
		}
	}
	return false
}

type imageURLsKey struct{}
type imageMarker struct{ token, url string }

func validateImageURL(value string) error {
	if strings.HasPrefix(value, "data:image/") {
		prefix, data, ok := strings.Cut(value, ";base64,")
		if !ok || (prefix != "data:image/png" && prefix != "data:image/jpeg" && prefix != "data:image/webp" && prefix != "data:image/gif") {
			return fmt.Errorf("unsupported image data URL")
		}
		if len(data) > 12*1024*1024 {
			return fmt.Errorf("image data URL exceeds 12 MiB encoded limit")
		}
		decoded, err := base64.StdEncoding.DecodeString(data)
		if err != nil || len(decoded) == 0 {
			return fmt.Errorf("invalid base64 image")
		}
		return nil
	}
	u, err := url.Parse(value)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil {
		return fmt.Errorf("image_url must be an HTTPS URL or supported base64 image data URL")
	}
	return nil
}

// Unique text markers carry image positions through the SDK's text-only message
// serializer. Only markers stored in this request's context can become images.
func prepareImageURLs(ctx context.Context, req llm.CompletionRequest, supported bool) (context.Context, llm.CompletionRequest, error) {
	if !RequestHasImages(req) {
		return ctx, req, nil
	}
	if !supported {
		return ctx, req, fmt.Errorf("selected model does not support image input")
	}
	req.Messages = append([]llm.Message(nil), req.Messages...)
	var markers []imageMarker
	for i, message := range req.Messages {
		req.Messages[i].Content = append([]llm.ContentBlock(nil), message.Content...)
		for j, block := range message.Content {
			if block.Type != imageURLBlockType {
				continue
			}
			if message.Role != llm.RoleUser {
				return ctx, req, fmt.Errorf("images must be supplied in user messages")
			}
			if err := validateImageURL(block.Text); err != nil {
				return ctx, req, err
			}
			marker := imageMarker{token: "[blacksand-image-" + uuid.NewString() + "]", url: block.Text}
			markers = append(markers, marker)
			req.Messages[i].Content[j] = llm.TextBlock(marker.token)
		}
	}
	return context.WithValue(ctx, imageURLsKey{}, markers), req, nil
}

type imageURLTransport struct{ inner http.RoundTripper }

func (t imageURLTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	markers, _ := request.Context().Value(imageURLsKey{}).([]imageMarker)
	if len(markers) == 0 {
		return t.inner.RoundTrip(request)
	}
	data, err := io.ReadAll(request.Body)
	request.Body.Close()
	if err != nil {
		return nil, err
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(data, &body); err != nil {
		return nil, err
	}
	var messages []map[string]json.RawMessage
	if err := json.Unmarshal(body["messages"], &messages); err != nil {
		return nil, err
	}
	for _, message := range messages {
		var text string
		if json.Unmarshal(message["content"], &text) != nil {
			continue
		}
		var parts []map[string]any
		remaining := text
		for {
			index := -1
			var next imageMarker
			for _, marker := range markers {
				if found := strings.Index(remaining, marker.token); found >= 0 && (index < 0 || found < index) {
					index, next = found, marker
				}
			}
			if index < 0 {
				break
			}
			if index > 0 {
				parts = append(parts, map[string]any{"type": "text", "text": remaining[:index]})
			}
			parts = append(parts, map[string]any{"type": "image_url", "image_url": map[string]string{"url": next.url}})
			remaining = remaining[index+len(next.token):]
		}
		if len(parts) == 0 {
			continue
		}
		if remaining != "" {
			parts = append(parts, map[string]any{"type": "text", "text": remaining})
		}
		message["content"], err = json.Marshal(parts)
		if err != nil {
			return nil, err
		}
	}
	body["messages"], err = json.Marshal(messages)
	if err != nil {
		return nil, err
	}
	data, err = json.Marshal(body)
	if err != nil {
		return nil, err
	}
	clone := request.Clone(request.Context())
	clone.Body = io.NopCloser(bytes.NewReader(data))
	clone.ContentLength = int64(len(data))
	clone.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(data)), nil }
	return t.inner.RoundTrip(clone)
}
