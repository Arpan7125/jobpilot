package agent

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/anthropics/anthropic-sdk-go/v2/option"
)

func TestTailorRequest(t *testing.T) {
	f := &fakeClaude{stopReason: "end_turn", text: `{"suggestions":[{"original":"a","rewritten":"b","reason":"c"}],"relevant":["a"]}`}
	newTestParser(t, f) // starts the fake server and sets f.url
	tl := NewClaudeTailorer(option.WithBaseURL(f.url), option.WithAPIKey("test-key"), option.WithMaxRetries(0))
	res, call, err := tl.Tailor(context.Background(), "my resume", "the posting", []string{"Go"})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Suggestions) != 1 || res.Relevant[0] != "a" || call.InputTokens != 812 {
		t.Errorf("res %+v call %+v", res, call)
	}
	oc, _ := f.body["output_config"].(map[string]any)
	if oc["effort"] != "medium" || f.body["fallbacks"] != "default" {
		t.Errorf("output_config %v fallbacks %v", oc, f.body["fallbacks"])
	}
	msgs, _ := json.Marshal(f.body["messages"])
	for _, want := range []string{"my resume", "the posting", "Skills the posting asks for: Go"} {
		if !strings.Contains(string(msgs), want) {
			t.Errorf("request missing %q", want)
		}
	}
}
