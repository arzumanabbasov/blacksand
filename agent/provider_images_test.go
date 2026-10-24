package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Autumn-27/norma/llm"
)

func TestGLMFlashImageWireContract(t *testing.T) {
	for _, streaming := range []bool{false, true} {
		t.Run(fmt.Sprint(streaming), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/v1/chat/completions" || r.Header.Get("Authorization") != "Bearer local-key" {
					t.Errorf("wrong endpoint/auth: %s", r.URL.Path)
				}
				var wire struct {
					Model     string `json:"model"`
					MaxTokens int    `json:"max_tokens"`
					Thinking  struct {
						Type string `json:"type"`
					} `json:"thinking"`
					Messages []struct {
						Role    string          `json:"role"`
						Content json.RawMessage `json:"content"`
					} `json:"messages"`
				}
				if err := json.NewDecoder(r.Body).Decode(&wire); err != nil {
					t.Error(err)
					return
				}
				if wire.Model != "zai-org/GLM-5.3-Flash" || wire.Thinking.Type != "enabled" || wire.MaxTokens != 500 {
					t.Errorf("wrong request: %+v", wire)
				}
				if len(wire.Messages) != 2 {
					t.Errorf("wrong message count: %d", len(wire.Messages))
					return
				}
				var parts []struct {
					Type  string `json:"type"`
					Text  string `json:"text"`
					Image struct {
						URL string `json:"url"`
					} `json:"image_url"`
				}
				if err := json.Unmarshal(wire.Messages[1].Content, &parts); err != nil {
					t.Error(err)
					return
				}
				if len(parts) != 4 || parts[0].Text != "Before" || parts[1].Image.URL != "https://example.com/image.jpg" || parts[2].Text != "After" || parts[3].Image.URL != "data:image/png;base64,aGVsbG8=" {
					t.Errorf("image order/URLs lost: %+v", parts)
				}
				if streaming {
					w.Header().Set("Content-Type", "text/event-stream")
					fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"A chart.\"},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n")
				} else {
					w.Header().Set("Content-Type", "application/json")
					fmt.Fprint(w, `{"choices":[{"message":{"role":"assistant","content":"A chart."},"finish_reason":"stop"}]}`)
				}
			}))
			defer server.Close()
			cfg := ConfigFrom("openai", "zai-org/GLM-5.3-Flash", server.URL+"/v1/chat/completions", "local-key", "")
			cfg.Retry = RetryConfig{ConnectAttempts: -1, EmptyAttempts: -1}
			provider, err := cfg.NewProvider()
			if err != nil {
				t.Fatal(err)
			}
			req := llm.CompletionRequest{System: []string{"Inspect"}, Messages: []llm.Message{{Role: llm.RoleUser, Content: []llm.ContentBlock{llm.TextBlock("Before"), ImageURLBlock("https://example.com/image.jpg"), llm.TextBlock("After"), ImageURLBlock("data:image/png;base64,aGVsbG8=")}}}, Thinking: "disabled", MaxTokens: 500}
			if streaming {
				var acc llm.Accumulator
				for event, err := range provider.Stream(t.Context(), req) {
					if err != nil {
						t.Fatal(err)
					}
					acc.Add(event)
				}
				if acc.Message().Text() != "A chart." {
					t.Fatal("image response lost")
				}
			} else {
				msg, _, _, err := provider.Complete(t.Context(), req)
				if err != nil || msg.Text() != "A chart." {
					t.Fatalf("response=%+v err=%v", msg, err)
				}
			}
			if req.Messages[0].Content[1].Type != imageURLBlockType || req.Thinking != "disabled" {
				t.Fatal("caller request was modified")
			}
		})
	}
}

func TestImageInputRejectedWithoutVision(t *testing.T) {
	req := llm.CompletionRequest{Messages: []llm.Message{{Role: llm.RoleUser, Content: []llm.ContentBlock{ImageURLBlock("https://example.com/image.jpg")}}}}
	if _, _, err := prepareImageURLs(context.Background(), req, false); err == nil {
		t.Fatal("image silently accepted by text-only model")
	}
	for _, input := range []string{"file:///secret", "https://user:password@example.com/image", "data:image/png;base64,invalid!", "data:text/plain;base64,aGVsbG8="} {
		if validateImageURL(input) == nil {
			t.Errorf("invalid image accepted: %s", input)
		}
	}
}
