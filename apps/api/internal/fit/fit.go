// Package fit scores a parsed posting against the user's profile. It is plain
// code, not a model call: the same inputs always give the same score, and
// every point can be traced to a reason.
package fit

import (
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/Arpan7125/jobpilot/apps/api/internal/agent"
	"github.com/Arpan7125/jobpilot/apps/api/internal/store"
)

// Result is the fit of one job. Score is null when neither side states enough
// to compare (for example an empty profile).
type Result struct {
	Score            *int     `json:"score"` // 0-100
	MatchedSkills    []string `json:"matchedSkills"`
	MissingSkills    []string `json:"missingSkills"`
	MatchedPreferred []string `json:"matchedPreferred"`
	Filters          []string `json:"filters"` // hard filters this job fails
	Reasons          []string `json:"reasons"`
}

// Weights of each part of the score. A part only counts when both the posting
// and the profile state it; the score is the weighted mean of the parts that count.
const (
	weightRequired   = 60
	weightPreferred  = 10
	weightExperience = 20
	weightLocation   = 10
)

// ExperienceSlack is how many years short of the posting's minimum the user
// can be before the job is filtered out.
const ExperienceSlack = 2

func Score(p agent.Posting, prof store.Profile) Result {
	r := Result{MatchedSkills: []string{}, MissingSkills: []string{}, MatchedPreferred: []string{}, Filters: []string{}, Reasons: []string{}}
	var total, weight float64
	add := func(w, v float64) { total += w * v; weight += w }

	have := map[string]bool{}
	for _, s := range prof.Skills {
		have[Normalize(s)] = true
	}

	if len(p.RequiredSkills) > 0 && len(have) > 0 {
		for _, s := range p.RequiredSkills {
			if have[Normalize(s)] {
				r.MatchedSkills = append(r.MatchedSkills, s)
			} else {
				r.MissingSkills = append(r.MissingSkills, s)
			}
		}
		add(weightRequired, float64(len(r.MatchedSkills))/float64(len(p.RequiredSkills)))
		r.Reasons = append(r.Reasons, fmt.Sprintf("You have %d of %d required skills.", len(r.MatchedSkills), len(p.RequiredSkills)))
	}

	if len(p.PreferredSkills) > 0 && len(have) > 0 {
		for _, s := range p.PreferredSkills {
			if have[Normalize(s)] {
				r.MatchedPreferred = append(r.MatchedPreferred, s)
			}
		}
		add(weightPreferred, float64(len(r.MatchedPreferred))/float64(len(p.PreferredSkills)))
		r.Reasons = append(r.Reasons, fmt.Sprintf("You have %d of %d nice-to-have skills.", len(r.MatchedPreferred), len(p.PreferredSkills)))
	}

	if p.Experience.Min != nil && prof.ExperienceYears != nil {
		gap := *p.Experience.Min - *prof.ExperienceYears
		switch {
		case gap <= 0:
			add(weightExperience, 1)
			r.Reasons = append(r.Reasons, fmt.Sprintf("You meet the %d+ years of experience asked for.", *p.Experience.Min))
		default:
			add(weightExperience, math.Max(0, 1-float64(gap)/float64(ExperienceSlack+1)))
			r.Reasons = append(r.Reasons, fmt.Sprintf("The posting asks for %d+ years; you have %d.", *p.Experience.Min, *prof.ExperienceYears))
			if gap > ExperienceSlack {
				r.Filters = append(r.Filters, fmt.Sprintf("Needs %d+ years of experience, you have %d.", *p.Experience.Min, *prof.ExperienceYears))
			}
		}
	}

	if v, reason, ok := location(p, prof); ok {
		add(weightLocation, v)
		r.Reasons = append(r.Reasons, reason)
	}

	if len(prof.WorkModes) > 0 && p.WorkMode != "" && p.WorkMode != "unknown" && !slices.Contains(prof.WorkModes, p.WorkMode) {
		r.Filters = append(r.Filters, fmt.Sprintf("Work mode is %s, you want %s.", p.WorkMode, strings.Join(prof.WorkModes, " or ")))
	}

	if f, note := payFilter(p.Pay, prof); f != "" {
		r.Filters = append(r.Filters, f)
	} else if note != "" {
		r.Reasons = append(r.Reasons, note)
	}

	if weight > 0 {
		s := int(math.Round(100 * total / weight))
		r.Score = &s
	} else {
		r.Reasons = append(r.Reasons, "Not enough in the posting or your profile to score this job.")
	}
	return r
}

// location scores 1 when the job is remote and the user accepts remote, or
// when one of the posting's locations names one of the user's cities.
func location(p agent.Posting, prof store.Profile) (float64, string, bool) {
	acceptsRemote := len(prof.WorkModes) == 0 || slices.Contains(prof.WorkModes, "remote")
	if p.WorkMode == "remote" && acceptsRemote {
		return 1, "The job is remote.", true
	}
	if len(p.Locations) == 0 || len(prof.Locations) == 0 {
		return 0, "", false
	}
	for _, loc := range p.Locations {
		for _, want := range prof.Locations {
			if w := strings.ToLower(strings.TrimSpace(want)); w != "" && strings.Contains(strings.ToLower(loc), w) {
				return 1, fmt.Sprintf("The job is in %s, one of your locations.", loc), true
			}
		}
	}
	return 0, fmt.Sprintf("The job is in %s, not one of your locations.", strings.Join(p.Locations, ", ")), true
}

// payFilter fails the job when its best stated pay is below the user's minimum.
// Pay in another currency or period is not converted, so it is never filtered.
func payFilter(pay agent.Pay, prof store.Profile) (filter, note string) {
	best := pay.Max
	if best == nil {
		best = pay.Min
	}
	if best == nil || prof.MinPay == nil {
		return "", ""
	}
	if pay.Currency == nil || prof.PayCurrency == nil || prof.PayPeriod == nil ||
		!strings.EqualFold(*pay.Currency, *prof.PayCurrency) || pay.Period != *prof.PayPeriod {
		return "", "Pay is in different units from your minimum, so it was not compared."
	}
	if *best < *prof.MinPay {
		return fmt.Sprintf("Pays up to %d %s per %s, below your minimum of %d.", *best, *pay.Currency, pay.Period, *prof.MinPay), ""
	}
	return "", fmt.Sprintf("Pay meets your minimum of %d %s per %s.", *prof.MinPay, *prof.PayCurrency, *prof.PayPeriod)
}

// aliases maps common spellings of a skill to one name, so "Golang" in a
// posting matches "Go" in a profile.
var aliases = map[string]string{
	"golang":              "go",
	"postgres":            "postgresql",
	"psql":                "postgresql",
	"k8s":                 "kubernetes",
	"js":                  "javascript",
	"ts":                  "typescript",
	"node":                "node.js",
	"nodejs":              "node.js",
	"node js":             "node.js",
	"reactjs":             "react",
	"react.js":            "react",
	"nextjs":              "next.js",
	"next":                "next.js",
	"py":                  "python",
	"amazon web services": "aws",
	"gcp":                 "google cloud",
	"c sharp":             "c#",
	"cpp":                 "c++",
	"rest":                "rest api",
	"rest apis":           "rest api",
	"restful apis":        "rest api",
}

// Normalize lowercases a skill and maps known aliases to one spelling.
func Normalize(skill string) string {
	s := strings.ToLower(strings.Join(strings.Fields(skill), " "))
	if a, ok := aliases[s]; ok {
		return a
	}
	return s
}
