package agent

import (
	"context"
	"iter"

	"github.com/Autumn-27/norma/llm"
)

// GLM-5.3 supports enabled thinking only, including auxiliary requests whose
// generic compaction/review callers disable it. Keep the configured effort and
// delegate to the same SDK provider so transport, retries, and rate limits stay
// unchanged. The pinned SDK exposes only Stream/Complete on these providers.
// https://docs.z.ai/guides/llm/glm-5.3
//
// Config.NewProvider applies this only to the exact model ID glm-5.3.
type glm53Provider struct {
	llm.Provider
	vision bool
}

func isGLM53Flash(model string) bool {
	return model == "zai-org/GLM-5.3-Flash" || model == "glm-5.3-flash"
}
func glm53RequiresThinking(model string) bool {
	return model == "glm-5.3" || model == "zai-org/GLM-5.3" || isGLM53Flash(model)
}
func (p glm53Provider) SupportsImageInput() bool { return p.vision }

func (p glm53Provider) Stream(ctx context.Context, req llm.CompletionRequest) iter.Seq2[llm.StreamEvent, error] {
	req.Thinking = "enabled"
	return func(yield func(llm.StreamEvent, error) bool) {
		ctx, req, err := prepareImageURLs(ctx, req, p.vision)
		if err != nil {
			yield(llm.StreamEvent{}, err)
			return
		}
		for event, err := range p.Provider.Stream(ctx, req) {
			if !yield(event, err) {
				return
			}
		}
	}
}

func (p glm53Provider) Complete(ctx context.Context, req llm.CompletionRequest) (llm.Message, string, llm.Usage, error) {
	req.Thinking = "enabled"
	ctx, req, err := prepareImageURLs(ctx, req, p.vision)
	if err != nil {
		return llm.Message{}, "", llm.Usage{}, err
	}
	return p.Provider.Complete(ctx, req)
}
