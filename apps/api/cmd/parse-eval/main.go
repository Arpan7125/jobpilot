// Command parse-eval runs the posting parser over real postings and scores it
// against hand-checked answers.
//
//	ANTHROPIC_API_KEY=... go run ./cmd/parse-eval [dir]
//
// For every NAME.txt in dir (default evals/postings) it writes the parser's
// output to NAME.out.json. If NAME.expected.json exists (same shape as the
// output, filled in by hand from the posting) it compares the two and prints
// a score per posting and in total.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/Arpan7125/jobpilot/apps/api/internal/agent"
)

func main() {
	dir := "evals/postings"
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	if err := run(dir); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type totals struct {
	postings, scored, failed  int
	fieldsRight, fields       int
	skillsFound, skillsWanted int // recall of required skills
	skillsRight, skillsGiven  int // precision of required skills
	inTokens, outTokens       int64
	latency                   time.Duration
}

func run(dir string) error {
	if os.Getenv("ANTHROPIC_API_KEY") == "" {
		return errors.New("ANTHROPIC_API_KEY is not set")
	}
	files, err := filepath.Glob(filepath.Join(dir, "*.txt"))
	if err != nil || len(files) == 0 {
		return fmt.Errorf("no postings (*.txt) in %s", dir)
	}
	parser := agent.NewClaudeParser()
	var t totals
	for _, path := range files {
		name := strings.TrimSuffix(filepath.Base(path), ".txt")
		text, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		t.postings++
		start := time.Now()
		got, call, err := parser.Parse(context.Background(), string(text))
		t.latency += time.Since(start)
		t.inTokens += call.InputTokens
		t.outTokens += call.OutputTokens
		if err != nil {
			t.failed++
			fmt.Printf("%-30s FAILED: %v\n", name, err)
			continue
		}
		out, _ := json.MarshalIndent(got, "", "  ")
		if err := os.WriteFile(filepath.Join(dir, name+".out.json"), append(out, '\n'), 0o644); err != nil {
			return err
		}

		raw, err := os.ReadFile(filepath.Join(dir, name+".expected.json"))
		if errors.Is(err, os.ErrNotExist) {
			fmt.Printf("%-30s parsed (no expected file yet)\n", name)
			continue
		} else if err != nil {
			return err
		}
		var want agent.Posting
		if err := json.Unmarshal(raw, &want); err != nil {
			return fmt.Errorf("%s.expected.json: %w", name, err)
		}
		t.scored++
		wrong := compare(&t, got, want)
		if len(wrong) == 0 {
			fmt.Printf("%-30s all fields match\n", name)
		} else {
			fmt.Printf("%-30s mismatched: %s\n", name, strings.Join(wrong, ", "))
		}
	}

	fmt.Printf("\n%d postings, %d failed, %d scored against expected answers\n", t.postings, t.failed, t.scored)
	if t.scored > 0 {
		fmt.Printf("fields correct:          %d/%d (%.0f%%)\n", t.fieldsRight, t.fields, pct(t.fieldsRight, t.fields))
		fmt.Printf("required skills recall:  %d/%d (%.0f%%)\n", t.skillsFound, t.skillsWanted, pct(t.skillsFound, t.skillsWanted))
		fmt.Printf("required skills precision: %d/%d (%.0f%%)\n", t.skillsRight, t.skillsGiven, pct(t.skillsRight, t.skillsGiven))
	}
	if n := t.postings; n > 0 {
		fmt.Printf("tokens: %d in, %d out; mean latency %s\n", t.inTokens, t.outTokens, (t.latency / time.Duration(n)).Round(time.Millisecond))
	}
	return nil
}

// compare scores the single-value fields exactly and the required skills as
// sets (case-insensitive). It returns the names of fields that differ.
func compare(t *totals, got, want agent.Posting) []string {
	var wrong []string
	check := func(name string, ok bool) {
		t.fields++
		if ok {
			t.fieldsRight++
		} else {
			wrong = append(wrong, name)
		}
	}
	check("seniority", got.Seniority == want.Seniority)
	check("employmentType", got.EmploymentType == want.EmploymentType)
	check("workMode", got.WorkMode == want.WorkMode)
	check("experienceYears", eqInt(got.Experience.Min, want.Experience.Min) && eqInt(got.Experience.Max, want.Experience.Max))
	check("pay", eqInt(got.Pay.Min, want.Pay.Min) && eqInt(got.Pay.Max, want.Pay.Max) &&
		eqStr(got.Pay.Currency, want.Pay.Currency) && got.Pay.Period == want.Pay.Period)

	gotSkills, wantSkills := lower(got.RequiredSkills), lower(want.RequiredSkills)
	t.skillsWanted += len(wantSkills)
	t.skillsGiven += len(gotSkills)
	for _, s := range wantSkills {
		if slices.Contains(gotSkills, s) {
			t.skillsFound++
			t.skillsRight++
		}
	}
	if !slices.Equal(gotSkills, wantSkills) {
		wrong = append(wrong, "requiredSkills")
	}
	return wrong
}

func lower(items []string) []string {
	out := make([]string, len(items))
	for i, s := range items {
		out[i] = strings.ToLower(strings.TrimSpace(s))
	}
	slices.Sort(out)
	return slices.Compact(out)
}

func eqInt(a, b *int) bool { return (a == nil && b == nil) || (a != nil && b != nil && *a == *b) }
func eqStr(a, b *string) bool {
	return (a == nil && b == nil) || (a != nil && b != nil && strings.EqualFold(*a, *b))
}

func pct(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return 100 * float64(a) / float64(b)
}
