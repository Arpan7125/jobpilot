package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go/v2"
	"github.com/anthropics/anthropic-sdk-go/v2/option"
)

// Suggestion is one proposed rewording of a resume line.
type Suggestion struct {
	Original  string `json:"original"`  // a full line copied from the resume
	Rewritten string `json:"rewritten"` // the same facts, worded for this job
	Reason    string `json:"reason"`
}

// TailorResult is the model's raw proposal, before the guards check it.
type TailorResult struct {
	Suggestions []Suggestion `json:"suggestions"`
	Relevant    []string     `json:"relevant"` // resume lines to lead with for this job
}

var tailorSchema = object(map[string]any{
	"suggestions": map[string]any{
		"type": "array",
		"items": object(map[string]any{
			"original":  map[string]any{"type": "string"},
			"rewritten": map[string]any{"type": "string"},
			"reason":    map[string]any{"type": "string"},
		}),
	},
	"relevant": stringList,
})

const tailorPrompt = `You help a job seeker adapt their resume to one job posting.

You may only rephrase or reorder what is already in the resume. Never add a skill, tool, language, framework, employer, title, degree, number, metric or claim that the resume line does not already state. If a line can't be improved without adding something, leave it out. It is fine to return few or no suggestions.

- suggestions: at most 8 resume lines worth rewording for this job. "original" must be one full line copied exactly from the resume, without its bullet character. "rewritten" keeps every fact of the original and only changes wording or emphasis, for example using the posting's name for a skill the line already mentions, or putting the part that matters to this job first. "reason" says in one short sentence why it helps for this job.
- relevant: up to 5 lines copied exactly from the resume that matter most for this job, most relevant first. The user may move them up.`

// Tailorer proposes rewordings of a resume for a posting.
type Tailorer interface {
	Tailor(ctx context.Context, resume, posting string, skills []string) (TailorResult, Call, error)
}

type ClaudeTailorer struct {
	client anthropic.Client
}

func NewClaudeTailorer(opts ...option.RequestOption) *ClaudeTailorer {
	return &ClaudeTailorer{client: anthropic.NewClient(opts...)}
}

func (t *ClaudeTailorer) Tailor(ctx context.Context, resume, posting string, skills []string) (TailorResult, Call, error) {
	user := "<resume>\n" + resume + "\n</resume>\n\n<posting>\n" + posting + "\n</posting>"
	if len(skills) > 0 {
		user += "\n\nSkills the posting asks for: " + strings.Join(skills, ", ")
	}
	resp, err := t.client.Beta.Messages.New(ctx, anthropic.BetaMessageNewParams{
		Model:     Model,
		MaxTokens: 8192,
		System:    []anthropic.BetaTextBlockParam{{Text: tailorPrompt}},
		Messages:  []anthropic.BetaMessageParam{anthropic.NewBetaUserMessage(anthropic.NewBetaTextBlock(user))},
		OutputConfig: anthropic.BetaOutputConfigParam{
			Effort: anthropic.BetaOutputConfigEffortMedium,
			Format: anthropic.BetaJSONOutputFormatParam{Schema: tailorSchema},
		},
		Fallbacks: anthropic.BetaFallbacksParamUnion{OfDefault: "default"},
		Betas:     []anthropic.AnthropicBeta{anthropic.AnthropicBetaServerSideFallback2026_07_01},
	})
	if err != nil {
		return TailorResult{}, Call{Model: Model}, fmt.Errorf("claude request: %w", err)
	}
	call := Call{Model: string(resp.Model), InputTokens: resp.Usage.InputTokens, OutputTokens: resp.Usage.OutputTokens}
	switch resp.StopReason {
	case anthropic.BetaStopReasonRefusal:
		return TailorResult{}, call, ErrRefused
	case anthropic.BetaStopReasonMaxTokens:
		return TailorResult{}, call, errors.New("claude response was cut off")
	}
	var text strings.Builder
	for _, block := range resp.Content {
		if b, ok := block.AsAny().(anthropic.BetaTextBlock); ok {
			text.WriteString(b.Text)
		}
	}
	var out TailorResult
	if err := json.Unmarshal([]byte(text.String()), &out); err != nil {
		return TailorResult{}, call, fmt.Errorf("decode tailor output: %w", err)
	}
	return out, call, nil
}
