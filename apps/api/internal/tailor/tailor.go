// Package tailor checks the model's resume suggestions before the user sees
// them and applies the ones the user approves. The prompt asks the model not
// to add anything; this package makes sure of it.
package tailor

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/Arpan7125/jobpilot/apps/api/internal/agent"
	"github.com/Arpan7125/jobpilot/apps/api/internal/fit"
)

// Statuses a suggestion moves through on the approval screen.
const (
	Pending  = "pending"
	Approved = "approved"
	Rejected = "rejected"
)

// Suggestion is a guarded suggestion awaiting the user's decision.
type Suggestion struct {
	agent.Suggestion
	Status string `json:"status"`
}

// Dropped is a suggestion the guards refused, kept so the user can see why.
type Dropped struct {
	agent.Suggestion
	Why string `json:"why"`
}

// MaxSuggestions caps what reaches the approval screen.
const MaxSuggestions = 8

var (
	bulletPrefix = regexp.MustCompile(`^\s*(?:[-*•▪◦·]|\d+[.)])?\s*`)
	numberRe     = regexp.MustCompile(`\d+(?:[.,]\d+)*`)
)

// key is how lines are compared: no bullet, single spaces.
func key(line string) string {
	return strings.Join(strings.Fields(bulletPrefix.ReplaceAllString(line, "")), " ")
}

// Guard keeps the suggestions that only reword an existing resume line, and
// drops any that quote a line the resume doesn't have, add a number the line
// doesn't have, or name a skill the resume doesn't mention. postingSkills are
// the skill names to watch for in addition to the built-in alias table.
func Guard(resume string, res agent.TailorResult, postingSkills []string) (kept []Suggestion, dropped []Dropped, relevant []string) {
	lines := map[string]bool{}
	for _, l := range strings.Split(resume, "\n") {
		if k := key(l); k != "" {
			lines[k] = true
		}
	}
	resumeSkills := skillsIn(resume, postingSkills)

	seen := map[string]bool{}
	kept, dropped = []Suggestion{}, []Dropped{}
	for _, s := range res.Suggestions {
		s.Original, s.Rewritten, s.Reason = key(s.Original), strings.TrimSpace(s.Rewritten), strings.TrimSpace(s.Reason)
		why := ""
		switch {
		case !lines[s.Original]:
			why = "It quotes a line that isn't in your resume."
		case s.Rewritten == "" || key(s.Rewritten) == s.Original:
			why = "It doesn't change the line."
		case seen[s.Original]:
			why = "Another suggestion already covers this line."
		default:
			if n := newNumbers(s.Original, s.Rewritten); len(n) > 0 {
				why = fmt.Sprintf("It adds %s, which the original line doesn't state.", strings.Join(n, ", "))
			} else if added := newSkills(s.Rewritten, resumeSkills, postingSkills); len(added) > 0 {
				why = fmt.Sprintf("It mentions %s, which your resume doesn't.", strings.Join(added, ", "))
			}
		}
		if why != "" {
			dropped = append(dropped, Dropped{Suggestion: s, Why: why})
			continue
		}
		if len(kept) == MaxSuggestions {
			continue
		}
		seen[s.Original] = true
		kept = append(kept, Suggestion{Suggestion: s, Status: Pending})
	}

	relevant = []string{}
	for _, l := range res.Relevant {
		if k := key(l); lines[k] && !slices.Contains(relevant, k) && len(relevant) < 5 {
			relevant = append(relevant, k)
		}
	}
	return kept, dropped, relevant
}

// newNumbers returns the numbers in rewritten that original doesn't contain.
func newNumbers(original, rewritten string) []string {
	have := numberRe.FindAllString(original, -1)
	var added []string
	for _, n := range numberRe.FindAllString(rewritten, -1) {
		if !slices.Contains(have, n) && !slices.Contains(added, n) {
			added = append(added, n)
		}
	}
	return added
}

// skillsIn returns the normalized names of the watched skills that text mentions.
func skillsIn(text string, extra []string) map[string]bool {
	found := map[string]bool{}
	for _, s := range append(fit.KnownSkills(), extra...) {
		if mentions(text, s) {
			found[fit.Normalize(s)] = true
		}
	}
	return found
}

// newSkills returns watched skills that rewritten names but the resume doesn't.
func newSkills(rewritten string, resumeSkills map[string]bool, extra []string) []string {
	var added []string
	for name := range skillsIn(rewritten, extra) {
		if !resumeSkills[name] {
			added = append(added, name)
		}
	}
	slices.Sort(added)
	return added
}

// mentions reports whether text contains term as a whole word, ignoring case.
// Terms like "C++" and "Node.js" keep their punctuation.
func mentions(text, term string) bool {
	term = strings.TrimSpace(term)
	if term == "" {
		return false
	}
	re := regexp.MustCompile(`(?i)(?:^|[^\pL\pN+#.])` + regexp.QuoteMeta(term) + `(?:$|[^\pL\pN+#])`)
	return re.MatchString(text)
}

// Apply returns the resume with each approved suggestion's line replaced,
// keeping the line's bullet and indentation.
func Apply(resume string, suggestions []Suggestion) string {
	approved := map[string]string{}
	for _, s := range suggestions {
		if s.Status == Approved {
			approved[s.Original] = s.Rewritten
		}
	}
	lines := strings.Split(resume, "\n")
	for i, l := range lines {
		if r, ok := approved[key(l)]; ok {
			lines[i] = bulletPrefix.FindString(l) + r
		}
	}
	return strings.Join(lines, "\n")
}
