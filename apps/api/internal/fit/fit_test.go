package fit

import (
	"strings"
	"testing"

	"github.com/Arpan7125/jobpilot/apps/api/internal/agent"
	"github.com/Arpan7125/jobpilot/apps/api/internal/store"
)

func ptr[T any](v T) *T { return &v }

func posting() agent.Posting {
	return agent.Posting{
		RequiredSkills:  []string{"Golang", "PostgreSQL", "Docker", "Kubernetes"},
		PreferredSkills: []string{"AWS", "Redis"},
		Experience:      agent.Range{Min: ptr(1)},
		Pay:             agent.Pay{Min: ptr(25000), Max: ptr(35000), Currency: ptr("INR"), Period: "month"},
		Locations:       []string{"Bengaluru, Karnataka"},
		WorkMode:        "hybrid",
	}
}

func profile() store.Profile {
	return store.Profile{
		Skills:          []string{"Go", "postgres", "Docker", "AWS"},
		ExperienceYears: ptr(1),
		Locations:       []string{"bengaluru"},
		MinPay:          ptr(20000), PayCurrency: ptr("INR"), PayPeriod: ptr("month"),
	}
}

func TestScoreBreakdown(t *testing.T) {
	r := Score(posting(), profile())
	// required 3/4 * 60 = 45, preferred 1/2 * 10 = 5, experience 20, location 10 → 80/100
	if r.Score == nil || *r.Score != 80 {
		t.Fatalf("score = %v, want 80; reasons %q", r.Score, r.Reasons)
	}
	if strings.Join(r.MatchedSkills, ",") != "Golang,PostgreSQL,Docker" || strings.Join(r.MissingSkills, ",") != "Kubernetes" {
		t.Errorf("matched %q missing %q", r.MatchedSkills, r.MissingSkills)
	}
	if strings.Join(r.MatchedPreferred, ",") != "AWS" {
		t.Errorf("preferred %q", r.MatchedPreferred)
	}
	if len(r.Filters) != 0 {
		t.Errorf("filters = %q, want none", r.Filters)
	}
}

func TestOnlyStatedPartsCount(t *testing.T) {
	// A posting with only required skills is scored on skills alone.
	p := agent.Posting{RequiredSkills: []string{"Go", "Rust"}}
	r := Score(p, store.Profile{Skills: []string{"go"}})
	if r.Score == nil || *r.Score != 50 {
		t.Errorf("score = %v, want 50", r.Score)
	}
	// An empty profile can't be scored.
	if r := Score(posting(), store.Profile{}); r.Score != nil {
		t.Errorf("empty profile score = %d, want null", *r.Score)
	}
}

func TestHardFilters(t *testing.T) {
	p := posting()
	p.Experience.Min = ptr(5)
	p.WorkMode = "onsite"
	p.Pay = agent.Pay{Max: ptr(15000), Currency: ptr("INR"), Period: "month"}
	prof := profile()
	prof.WorkModes = []string{"remote", "hybrid"}
	r := Score(p, prof)
	if len(r.Filters) != 3 {
		t.Fatalf("filters = %q, want experience, work mode and pay", r.Filters)
	}
}

func TestNeverFilterOnUnknowns(t *testing.T) {
	p := posting()
	p.WorkMode = "unknown"
	p.Experience = agent.Range{}
	p.Pay = agent.Pay{Min: ptr(1000), Currency: ptr("USD"), Period: "month"} // different currency
	prof := profile()
	prof.WorkModes = []string{"remote"}
	r := Score(p, prof)
	if len(r.Filters) != 0 {
		t.Errorf("filters = %q, want none for unknown or unconvertible values", r.Filters)
	}
}

func TestExperienceWithinSlack(t *testing.T) {
	p := posting()
	p.Experience.Min = ptr(3) // 2 years short: lower score, not filtered
	r := Score(p, profile())
	if len(r.Filters) != 0 {
		t.Errorf("filters = %q", r.Filters)
	}
	if *r.Score >= 80 {
		t.Errorf("score = %d, want lower than a full experience match", *r.Score)
	}
}

func TestRemoteCountsAsLocationMatch(t *testing.T) {
	p := posting()
	p.WorkMode, p.Locations = "remote", []string{"Pune"}
	r := Score(p, profile())
	if *r.Score != 80 {
		t.Errorf("score = %d, want 80 (remote job, user accepts any mode)", *r.Score)
	}
}

func TestNormalize(t *testing.T) {
	for in, want := range map[string]string{"Golang": "go", " Node JS ": "node.js", "NodeJS": "node.js", "K8s": "kubernetes", "Next.js": "next.js"} {
		if got := Normalize(in); got != want {
			t.Errorf("Normalize(%q) = %q, want %q", in, got, want)
		}
	}
}
