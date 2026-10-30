package httpapi

import (
	"context"
	"strings"
	"testing"

	"github.com/Arpan7125/jobpilot/apps/api/internal/agent"
)

type fakeTailorer struct {
	res    agent.TailorResult
	skills []string
}

func (f *fakeTailorer) Tailor(_ context.Context, _, _ string, skills []string) (agent.TailorResult, agent.Call, error) {
	f.skills = skills
	return f.res, agent.Call{Model: "fake"}, nil
}

const testResume = "Projects\n- Built a job tracker API in Golang with PostgreSQL\n- Wrote tests against a real database"

func TestTailoring(t *testing.T) {
	ft := &fakeTailorer{res: agent.TailorResult{
		Suggestions: []agent.Suggestion{
			{Original: "Built a job tracker API in Golang with PostgreSQL", Rewritten: "Built a Go and PostgreSQL API for a job tracker", Reason: "Leads with the stack"},
			{Original: "Wrote tests against a real database", Rewritten: "Wrote Kubernetes tests against a real database", Reason: "Adds k8s"},
		},
		Relevant: []string{"Built a job tracker API in Golang with PostgreSQL"},
	}}
	fp := &fakeParser{result: agent.Posting{RequiredSkills: []string{"Go", "Kubernetes"}}}
	c := signedIn(t)
	c.h = newClientWithAI(t, fp, ft).h

	var job map[string]any
	c.do("POST", "/jobs", validJob(), &job)
	id := job["id"].(string)

	var errBody map[string]any
	if code := c.do("POST", "/jobs/"+id+"/tailor", nil, &errBody); code != 422 {
		t.Fatalf("tailor without resume = %d %v", code, errBody)
	}
	c.do("PUT", "/profile", map[string]any{"resume": testResume}, nil)
	c.do("POST", "/jobs/"+id+"/parse", nil, nil)

	var got map[string]any
	if code := c.do("GET", "/jobs/"+id+"/tailoring", nil, &got); code != 200 || got != nil {
		t.Fatalf("no tailoring yet = %d %v", code, got)
	}
	if code := c.do("POST", "/jobs/"+id+"/tailor", nil, &got); code != 201 {
		t.Fatalf("tailor = %d %v", code, got)
	}
	if strings.Join(ft.skills, ",") != "Go,Kubernetes" {
		t.Errorf("tailorer got skills %q", ft.skills)
	}
	sugs, dropped := got["suggestions"].([]any), got["dropped"].([]any)
	if len(sugs) != 1 || len(dropped) != 1 {
		t.Fatalf("suggestions %v dropped %v", sugs, dropped)
	}
	if got["tailored"] != testResume {
		t.Errorf("nothing approved yet, tailored = %q", got["tailored"])
	}

	if code := c.do("PUT", "/jobs/"+id+"/tailoring/suggestions/0", map[string]string{"status": "approved"}, &got); code != 200 {
		t.Fatalf("approve = %d %v", code, got)
	}
	if !strings.Contains(got["tailored"].(string), "- Built a Go and PostgreSQL API for a job tracker") {
		t.Errorf("tailored = %q", got["tailored"])
	}
	if code := c.do("PUT", "/jobs/"+id+"/tailoring/suggestions/5", map[string]string{"status": "approved"}, nil); code != 404 {
		t.Errorf("bad index = %d", code)
	}
	if code := c.do("PUT", "/jobs/"+id+"/tailoring/suggestions/0", map[string]string{"status": "maybe"}, nil); code != 422 {
		t.Errorf("bad status = %d", code)
	}

	// Editing the resume marks the suggestions stale.
	c.do("PUT", "/profile", map[string]any{"resume": testResume + "\n- New line"}, nil)
	c.do("GET", "/jobs/"+id+"/tailoring", nil, &got)
	if got["stale"] != true {
		t.Error("tailoring not marked stale after resume edit")
	}

	other := signedIn(t)
	other.h = c.h
	if code := other.do("GET", "/jobs/"+id+"/tailoring", nil, nil); code != 404 {
		t.Errorf("other user = %d", code)
	}
	if code := other.do("PUT", "/jobs/"+id+"/tailoring/suggestions/0", map[string]string{"status": "rejected"}, nil); code != 404 {
		t.Errorf("other user decide = %d", code)
	}
}
