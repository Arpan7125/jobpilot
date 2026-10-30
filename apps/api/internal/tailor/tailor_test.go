package tailor

import (
	"strings"
	"testing"

	"github.com/Arpan7125/jobpilot/apps/api/internal/agent"
)

const resume = `Arpan Mukherjee
Projects
- Built a job tracker API in Golang with PostgreSQL and JWT auth
- Wrote integration tests that run against a real database
  • Cut page load time by 40% by caching queries`

func sug(original, rewritten string) agent.Suggestion {
	return agent.Suggestion{Original: original, Rewritten: rewritten, Reason: "fits the job"}
}

func TestGuard(t *testing.T) {
	res := agent.TailorResult{
		Suggestions: []agent.Suggestion{
			sug("Built a job tracker API in Golang with PostgreSQL and JWT auth", "Built a job tracker API in Go with PostgreSQL and JWT authentication"), // ok: Go = Golang
			sug("Wrote integration tests that run against a real database", "Wrote integration tests in Docker against a real database"),                  // adds Docker
			sug("Cut page load time by 40% by caching queries", "Cut page load time by 60% by caching queries"),                                           // changes the number
			sug("Led a team of five engineers", "Led five engineers"),                                                                                     // not in resume
			sug("  -  Wrote integration tests   that run against a real database", "Wrote integration tests against a live Postgres database"),            // ok: Postgres is in the resume
			sug("Built a job tracker API in Golang with PostgreSQL and JWT auth", "Same line, again"),                                                     // duplicate
			sug("Cut page load time by 40% by caching queries", "Cut page load time by 40% by caching queries"),                                           // unchanged
		},
		Relevant: []string{"Wrote integration tests that run against a real database", "Something invented"},
	}
	kept, dropped, relevant := Guard(resume, res, []string{"Kubernetes", "Docker"})
	if len(kept) != 2 {
		t.Fatalf("kept %d: %+v", len(kept), kept)
	}
	for _, k := range kept {
		if k.Status != Pending {
			t.Errorf("status = %q", k.Status)
		}
	}
	whys := []string{}
	for _, d := range dropped {
		whys = append(whys, d.Why)
	}
	all := strings.Join(whys, " | ")
	for _, want := range []string{"mentions docker", "adds 60", "isn't in your resume", "already covers", "doesn't change"} {
		if !strings.Contains(all, want) {
			t.Errorf("no drop reason containing %q in %q", want, all)
		}
	}
	if len(relevant) != 1 {
		t.Errorf("relevant = %q", relevant)
	}
}

func TestMentionsIsWholeWord(t *testing.T) {
	cases := map[string]bool{
		"Used Go daily":      true,
		"Used Golang":        false, // "go" alone isn't in "golang" as a word
		"Wrote C++ services": true,
		"Node.js backend":    true,
		"googled things":     false,
	}
	for text, want := range cases {
		term := "go"
		if strings.Contains(text, "C++") {
			term = "c++"
		} else if strings.Contains(text, "Node") {
			term = "node.js"
		}
		if got := mentions(text, term); got != want {
			t.Errorf("mentions(%q, %q) = %v", text, term, got)
		}
	}
}

func TestApply(t *testing.T) {
	s := []Suggestion{
		{Suggestion: sug("Cut page load time by 40% by caching queries", "Cached queries to cut page load time by 40%"), Status: Approved},
		{Suggestion: sug("Wrote integration tests that run against a real database", "nope"), Status: Rejected},
	}
	got := Apply(resume, s)
	if !strings.Contains(got, "  • Cached queries to cut page load time by 40%") {
		t.Errorf("approved line not applied with its bullet:\n%s", got)
	}
	if !strings.Contains(got, "- Wrote integration tests that run against a real database") {
		t.Errorf("rejected line changed:\n%s", got)
	}
}

func TestGuardDropsUnstatedQualifiers(t *testing.T) {
	// "REST API" is a known skill; the resume only says "API".
	res := agent.TailorResult{Suggestions: []agent.Suggestion{
		sug("Built a job tracker API in Golang with PostgreSQL and JWT auth", "Built a REST API in Go with PostgreSQL"),
	}}
	kept, dropped, _ := Guard(resume, res, nil)
	if len(kept) != 0 || len(dropped) != 1 || !strings.Contains(dropped[0].Why, "rest api") {
		t.Errorf("kept %+v dropped %+v", kept, dropped)
	}
}

func TestGuardWithoutPostingSkills(t *testing.T) {
	// The watch list catches common skills even when the job wasn't parsed.
	res := agent.TailorResult{Suggestions: []agent.Suggestion{
		sug("Wrote integration tests that run against a real database", "Wrote integration tests in Docker against a real database"),
	}}
	kept, _, _ := Guard(resume, res, nil)
	if len(kept) != 0 {
		t.Errorf("kept %+v, want Docker caught without posting skills", kept)
	}
}
