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

func TestProfileAndFit(t *testing.T) {
	min := 0
	fake := &fakeParser{result: agent.Posting{
		RequiredSkills: []string{"Go", "PostgreSQL"}, Experience: agent.Range{Min: &min}, WorkMode: "onsite",
	}}
	c := signedIn(t)
	c.h = newClientWithParser(t, fake).h

	var prof map[string]any
	if code := c.do("GET", "/profile", nil, &prof); code != 200 || prof["updatedAt"] != nil {
		t.Fatalf("empty profile = %d %v", code, prof)
	}
	var verr struct{ Fields map[string]string }
	if code := c.do("PUT", "/profile", map[string]any{"workModes": []string{"moon"}, "minPay": 100}, &verr); code != 422 ||
		verr.Fields["workModes"] == "" || verr.Fields["minPay"] == "" {
		t.Errorf("bad profile = %d %v", code, verr.Fields)
	}
	body := map[string]any{"skills": []string{" golang ", "Go", "Docker", ""}, "experienceYears": 0, "workModes": []string{"Remote"}}
	if code := c.do("PUT", "/profile", body, &prof); code != 200 {
		t.Fatalf("save profile = %d %v", code, prof)
	}
	if skills := prof["skills"].([]any); len(skills) != 3 { // trimmed, empty dropped; "Go" differs from "golang" as text
		t.Errorf("skills = %v", skills)
	}

	var job map[string]any
	c.do("POST", "/jobs", validJob(), &job)
	if job["fit"] != nil {
		t.Errorf("unparsed job fit = %v, want null", job["fit"])
	}
	id := job["id"].(string)
	c.do("POST", "/jobs/"+id+"/parse", nil, &job)
	f := job["fit"].(map[string]any)
	// 1 of 2 required skills (30/60) + experience met (20/20) = 50/80 → 63
	if f["score"] != float64(63) {
		t.Errorf("fit = %v", f)
	}
	if filters := f["filters"].([]any); len(filters) != 1 { // onsite job, user wants remote
		t.Errorf("filters = %v", filters)
	}
	var list []map[string]any
	c.do("GET", "/jobs", nil, &list)
	if list[0]["fit"] == nil {
		t.Error("list is missing fit")
	}
}
