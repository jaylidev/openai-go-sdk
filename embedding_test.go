package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEmbeddingBuilder_Do(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/embeddings" {
			t.Errorf("expected /v1/embeddings, got %s", r.URL.Path)
		}
		resp := EmbeddingResponse{
			Object: "list",
			Data: []EmbeddingData{
				{Object: "embedding", Index: 0, Embedding: []float64{0.1, 0.2, 0.3}},
			},
			Model: "deepseek-chat",
			Usage: Usage{PromptTokens: 3, TotalTokens: 3},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := NewClient(
		WithModel(DeepSeekChat),
		WithAPIKey("sk-test"),
		WithCustomBaseURL(srv.URL),
	)

	resp, err := client.Embedding().
		Input([]string{"你好", "世界"}).
		Do(context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Errorf("expected 1 data, got %d", len(resp.Data))
	}
	if resp.Data[0].Embedding[0] != 0.1 {
		t.Errorf("unexpected first value: %f", resp.Data[0].Embedding[0])
	}
}
