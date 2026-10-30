// Package agent holds JobPilot's LLM features. Week 2 starts with the posting
// parser: raw posting text in, structured fields out.
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

const Model = "claude-opus-5-5"

// ErrRefused means the model declined the request.
var ErrRefused = errors.New("model declined to parse the posting")

// Posting is what the parser extracts. Fields the posting doesn't state are
// null or empty: the parser must never guess.
type Posting struct {
	Role            string   `json:"role"`
	Company         *string  `json:"company"`
	Seniority       string   `json:"seniority"`       // intern, entry, mid, senior, lead, unknown
	EmploymentType  string   `json:"employmentType"`  // full_time, part_time, internship, contract, unknown
	RequiredSkills  []string `json:"requiredSkills"`  // as written in the posting
	PreferredSkills []string `json:"preferredSkills"` // "nice to have", "bonus", "plus"
	Experience      Range    `json:"experienceYears"`
	Pay             Pay      `json:"pay"`
	Locations       []string `json:"locations"`
	WorkMode        string   `json:"workMode"` // onsite, hybrid, remote, unknown
	Degree          *string  `json:"degree"`   // e.g. "B.Tech in CS or related", null if not stated
}

type Range struct {
	Min *int `json:"min"`
	Max *int `json:"max"`
}

type Pay struct {
	Min      *int    `json:"min"`
	Max      *int    `json:"max"`
	Currency *string `json:"currency"` // ISO 4217
	Period   string  `json:"period"`   // hour, month, year, total, unknown
}

func nullable(t string) map[string]any { return map[string]any{"type": []string{t, "null"}} }

func enum(values ...string) map[string]any { return map[string]any{"type": "string", "enum": values} }

func object(props map[string]any) map[string]any {
	required := make([]string, 0, len(props))
	for k := range props {
		required = append(required, k)
	}
	return map[string]any{"type": "object", "properties": props, "required": required, "additionalProperties": false}
}

var stringList = map[string]any{"type": "array", "items": map[string]any{"type": "string"}}

// postingSchema mirrors Posting. Structured outputs guarantee the response matches it.
var postingSchema = object(map[string]any{
	"role":            map[string]any{"type": "string"},
	"company":         nullable("string"),
	"seniority":       enum("intern", "entry", "mid", "senior", "lead", "unknown"),
	"employmentType":  enum("full_time", "part_time", "internship", "contract", "unknown"),
	"requiredSkills":  stringList,
	"preferredSkills": stringList,
	"experienceYears": object(map[string]any{"min": nullable("integer"), "max": nullable("integer")}),
	"pay": object(map[string]any{
		"min":      nullable("integer"),
		"max":      nullable("integer"),
		"currency": nullable("string"),
		"period":   enum("hour", "month", "year", "total", "unknown"),
	}),
	"locations": stringList,
	"workMode":  enum("onsite", "hybrid", "remote", "unknown"),
	"degree":    nullable("string"),
})

const systemPrompt = `You extract structured data from job postings for a job-application tracker.

Report only what the posting states. If a field is not stated, use null, an empty list, or "unknown". Do not infer pay from the company or role, do not infer experience from the title, and do not add skills the posting does not name.

- requiredSkills: concrete skills, tools, languages and frameworks the posting requires, in the posting's own wording (e.g. "Go", "PostgreSQL", "Docker"). One skill per entry. Skip soft skills such as "communication".
- preferredSkills: skills marked as nice to have, a plus, bonus or preferred.
- experienceYears: the stated range. "2+ years" is min 2, max null. "Freshers" or "0-1 years" is min 0.
- pay: numbers as written, converted to whole units (e.g. "20k" is 20000, "12 LPA" is 1200000 with period "year"). Stipends are pay. Currency as an ISO code (INR, USD).
- seniority: from explicit words only (intern, junior/entry, senior, lead). Otherwise "unknown".
- locations: cities or regions as written. workMode from explicit words (remote, hybrid, on-site/in office).`

// Call describes one model request, for the model-call log.
type Call struct {
	Model        string // the model that answered (a fallback model if one was used)
	InputTokens  int64
	OutputTokens int64
}

// Parser turns posting text into a Posting.
type Parser interface {
	Parse(ctx context.Context, posting string) (Posting, Call, error)
}

type ClaudeParser struct {
	client anthropic.Client
}

// NewClaudeParser builds a parser. Extra options (API key, base URL) are passed
// to the Anthropic client; by default it reads ANTHROPIC_API_KEY.
func NewClaudeParser(opts ...option.RequestOption) *ClaudeParser {
	return &ClaudeParser{client: anthropic.NewClient(opts...)}
}

func (p *ClaudeParser) Parse(ctx context.Context, posting string) (Posting, Call, error) {
	resp, err := p.client.Beta.Messages.New(ctx, anthropic.BetaMessageNewParams{
		Model:     Model,
		MaxTokens: 4096,
		System:    []anthropic.BetaTextBlockParam{{Text: systemPrompt}},
		Messages: []anthropic.BetaMessageParam{
			anthropic.NewBetaUserMessage(anthropic.NewBetaTextBlock("<posting>\n" + posting + "\n</posting>")),
		},
		OutputConfig: anthropic.BetaOutputConfigParam{
			// Extraction is not hard reasoning; low effort keeps it fast and cheap.
			Effort: anthropic.BetaOutputConfigEffortLow,
			Format: anthropic.BetaJSONOutputFormatParam{Schema: postingSchema},
		},
		// If the model declines, the API retries on a fallback model in the same call.
		Fallbacks: anthropic.BetaFallbacksParamUnion{OfDefault: "default"},
		Betas:     []anthropic.AnthropicBeta{anthropic.AnthropicBetaServerSideFallback2026_07_01},
	})
	if err != nil {
		return Posting{}, Call{Model: Model}, fmt.Errorf("claude request: %w", err)
	}
	call := Call{Model: string(resp.Model), InputTokens: resp.Usage.InputTokens, OutputTokens: resp.Usage.OutputTokens}
	switch resp.StopReason {
	case anthropic.BetaStopReasonRefusal:
		return Posting{}, call, ErrRefused
	case anthropic.BetaStopReasonMaxTokens:
		return Posting{}, call, errors.New("claude response was cut off")
	}

	var text strings.Builder
	for _, block := range resp.Content {
		if b, ok := block.AsAny().(anthropic.BetaTextBlock); ok {
			text.WriteString(b.Text)
		}
	}
	var out Posting
	if err := json.Unmarshal([]byte(text.String()), &out); err != nil {
		return Posting{}, call, fmt.Errorf("decode parser output: %w", err)
	}
	return out.normalized(), call, nil
}

// normalized cleans up the model output: trims and de-duplicates skills and
// drops impossible ranges, so downstream code can trust the shape.
func (p Posting) normalized() Posting {
	p.Role = strings.TrimSpace(p.Role)
	p.RequiredSkills = dedupe(p.RequiredSkills)
	p.PreferredSkills = dedupe(p.PreferredSkills)
	// A skill listed as required shouldn't also appear as preferred.
	required := map[string]bool{}
	for _, s := range p.RequiredSkills {
		required[strings.ToLower(s)] = true
	}
	preferred := p.PreferredSkills[:0]
	for _, s := range p.PreferredSkills {
		if !required[strings.ToLower(s)] {
			preferred = append(preferred, s)
		}
	}
	p.PreferredSkills = preferred
	p.Locations = dedupe(p.Locations)
	p.Experience = validRange(p.Experience)
	r := validRange(Range{p.Pay.Min, p.Pay.Max})
	p.Pay.Min, p.Pay.Max = r.Min, r.Max
	if p.Pay.Currency != nil {
		c := strings.ToUpper(strings.TrimSpace(*p.Pay.Currency))
		if len(c) == 3 {
			p.Pay.Currency = &c
		} else {
			p.Pay.Currency = nil
		}
	}
	return p
}

func dedupe(items []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, s := range items {
		s = strings.TrimSpace(s)
		if s == "" || seen[strings.ToLower(s)] {
			continue
		}
		seen[strings.ToLower(s)] = true
		out = append(out, s)
	}
	return out
}

func validRange(r Range) Range {
	if r.Min != nil && *r.Min < 0 {
		r.Min = nil
	}
	if r.Max != nil && *r.Max < 0 {
		r.Max = nil
	}
	if r.Min != nil && r.Max != nil && *r.Min > *r.Max {
		r.Min, r.Max = r.Max, r.Min
	}
	return r
}
