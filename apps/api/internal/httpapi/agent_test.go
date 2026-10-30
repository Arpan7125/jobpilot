package httpapi

import (
	"context"
	"testing"

	"github.com/Arpan7125/jobpilot/apps/api/internal/agent"
)

type fakeParser struct {
	got    string
	result agent.Posting
	err    error
}

func (f *fakeParser) Parse(_ context.Context, posting string) (agent.Posting, agent.Call, error) {
	f.got = posting
	return f.result, agent.Call{Model: "fake", InputTokens: 100, OutputTokens: 20}, f.err
}

func TestParseJob(t *testing.T) {
	company := "Acme"
	fake := &fakeParser{result: agent.Posting{Role: "Backend Intern", Company: &company, RequiredSkills: []string{"Go"}}}
	c := signedIn(t)
	c.h = newClientWithParser(t, fake).h

	var job map[string]any
	if code := c.do("POST", "/jobs", validJob(), &job); code != 201 {
		t.Fatalf("create = %d", code)
	}
	if job["parsed"] != nil {
		t.Fatalf("new job parsed = %v, want null", job["parsed"])
	}
	id := job["id"].(string)

	if code := c.do("POST", "/jobs/"+id+"/parse", nil, &job); code != 200 {
		t.Fatalf("parse = %d %v", code, job)
	}
	if fake.got != "Go, Postgres, Docker" {
		t.Errorf("parser got %q, want the job description", fake.got)
	}
	parsed := job["parsed"].(map[string]any)
	if parsed["role"] != "Backend Intern" || job["parsedAt"] == nil {
		t.Errorf("parsed = %v, parsedAt = %v", parsed, job["parsedAt"])
	}

	// The call is logged with its token counts.
	var n, in int
	if err := testPool.QueryRow(context.Background(),
		`SELECT count(*), coalesce(sum(input_tokens), 0) FROM model_calls WHERE job_id = $1 AND ok`, id).Scan(&n, &in); err != nil {
		t.Fatal(err)
	}
	if n != 1 || in != 100 {
		t.Errorf("model_calls = %d rows, %d input tokens", n, in)
	}

	// A refusal is a 422 and is logged as a failed call; the old result stays.
	fake.err = agent.ErrRefused
	if code := c.do("POST", "/jobs/"+id+"/parse", nil, nil); code != 422 {
		t.Errorf("refused parse = %d, want 422", code)
	}
	c.do("GET", "/jobs/"+id, nil, &job)
	if job["parsed"] == nil {
		t.Error("refusal wiped the earlier parse")
	}

	// Another user can't parse this job.
	other := signedIn(t)
	other.h = c.h
	if code := other.do("POST", "/jobs/"+id+"/parse", nil, nil); code != 404 {
		t.Errorf("other user parse = %d, want 404", code)
	}
}

func TestParseWithoutKey(t *testing.T) {
	c := signedIn(t)
	var job map[string]any
	c.do("POST", "/jobs", validJob(), &job)
	if code := c.do("POST", "/jobs/"+job["id"].(string)+"/parse", nil, nil); code != 503 {
		t.Errorf("parse without parser = %d, want 503", code)
	}
}
