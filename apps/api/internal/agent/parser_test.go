package agent

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/anthropics/anthropic-sdk-go/v2/option"
)

// fakeClaude answers like the Messages API and records the last request.
type fakeClaude struct {
	stopReason string
	text       string
	body       map[string]any
	header     http.Header
	url        string
}

func (f *fakeClaude) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	raw, _ := io.ReadAll(r.Body)
	json.Unmarshal(raw, &f.body)
	f.header = r.Header.Clone()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"id": "msg_test", "type": "message", "role": "assistant", "model": Model,
		"content":     []map[string]any{{"type": "text", "text": f.text}},
		"stop_reason": f.stopReason, "stop_sequence": nil,
		"usage": map[string]any{"input_tokens": 812, "output_tokens": 95},
	})
}

func newTestParser(t *testing.T, f *fakeClaude) *ClaudeParser {
	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)
	f.url = srv.URL
	return NewClaudeParser(option.WithBaseURL(srv.URL), option.WithAPIKey("test-key"), option.WithMaxRetries(0))
}

const sampleOutput = `{
  "role": " Backend Engineer Intern ",
  "company": "Acme",
  "seniority": "intern",
  "employmentType": "internship",
  "requiredSkills": ["Go", "PostgreSQL", "go", " Docker ", ""],
  "preferredSkills": ["Kubernetes", "docker"],
  "experienceYears": {"min": 0, "max": null},
  "pay": {"min": 40000, "max": 25000, "currency": "inr", "period": "month"},
  "locations": ["Bengaluru"],
  "workMode": "hybrid",
  "degree": null
}`

func TestParseSendsStructuredRequest(t *testing.T) {
	f := &fakeClaude{stopReason: "end_turn", text: sampleOutput}
	got, call, err := newTestParser(t, f).Parse(context.Background(), "We need a Go intern.")
	if err != nil {
		t.Fatal(err)
	}

	if f.body["model"] != Model {
		t.Errorf("model = %v", f.body["model"])
	}
	oc, _ := f.body["output_config"].(map[string]any)
	format, _ := oc["format"].(map[string]any)
	if format["type"] != "json_schema" || format["schema"] == nil || oc["effort"] != "low" {
		t.Errorf("output_config = %v", oc)
	}
	if f.body["fallbacks"] != "default" {
		t.Errorf("fallbacks = %v", f.body["fallbacks"])
	}
	if !strings.Contains(f.header.Get("Anthropic-Beta"), "server-side-fallback") {
		t.Errorf("anthropic-beta = %q", f.header.Get("Anthropic-Beta"))
	}
	msgs, _ := json.Marshal(f.body["messages"])
	if !strings.Contains(string(msgs), "We need a Go intern.") {
		t.Errorf("posting missing from messages: %s", msgs)
	}

	if call.Model != Model || call.InputTokens != 812 || call.OutputTokens != 95 {
		t.Errorf("call = %+v", call)
	}
	if got.Role != "Backend Engineer Intern" {
		t.Errorf("role = %q", got.Role)
	}
	if strings.Join(got.RequiredSkills, ",") != "Go,PostgreSQL,Docker" {
		t.Errorf("required = %q", got.RequiredSkills)
	}
	// Docker is already required, so it drops out of preferred.
	if strings.Join(got.PreferredSkills, ",") != "Kubernetes" {
		t.Errorf("preferred = %q", got.PreferredSkills)
	}
	if *got.Pay.Min != 25000 || *got.Pay.Max != 40000 || *got.Pay.Currency != "INR" {
		t.Errorf("pay = %d-%d %s", *got.Pay.Min, *got.Pay.Max, *got.Pay.Currency)
	}
	if got.Experience.Min == nil || *got.Experience.Min != 0 || got.Experience.Max != nil {
		t.Errorf("experience = %+v", got.Experience)
	}
}

func TestParseRefusal(t *testing.T) {
	f := &fakeClaude{stopReason: "refusal", text: ""}
	_, _, err := newTestParser(t, f).Parse(context.Background(), "posting")
	if !errors.Is(err, ErrRefused) {
		t.Fatalf("err = %v, want ErrRefused", err)
	}
}

func TestParseCutOff(t *testing.T) {
	f := &fakeClaude{stopReason: "max_tokens", text: `{"role":`}
	if _, _, err := newTestParser(t, f).Parse(context.Background(), "posting"); err == nil {
		t.Fatal("want an error for a truncated response")
	}
}

func TestSchemaRequiresEveryField(t *testing.T) {
	// Structured outputs need every property listed as required.
	props := postingSchema["properties"].(map[string]any)
	if len(postingSchema["required"].([]string)) != len(props) {
		t.Fatalf("required %v does not cover %d properties", postingSchema["required"], len(props))
	}
	var p Posting
	raw, _ := json.Marshal(p)
	var fields map[string]any
	json.Unmarshal(raw, &fields)
	for k := range fields {
		if _, ok := props[k]; !ok {
			t.Errorf("Posting field %q missing from schema", k)
		}
	}
}
