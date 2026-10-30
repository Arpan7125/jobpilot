package httpapi

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/Arpan7125/jobpilot/apps/api/internal/agent"
	"github.com/Arpan7125/jobpilot/apps/api/internal/fit"
	"github.com/Arpan7125/jobpilot/apps/api/internal/store"
)

var (
	workModes  = []string{"onsite", "hybrid", "remote"}
	payPeriods = []string{"hour", "month", "year", "total"}
)

func (s *Server) getProfile(w http.ResponseWriter, r *http.Request) {
	p, err := s.store.Profile(r.Context(), userID(r))
	if err != nil {
		serverError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// putProfile replaces the profile. Skills are stored exactly as the user
// typed them (trimmed and de-duplicated); nothing is added.
func (s *Server) putProfile(w http.ResponseWriter, r *http.Request) {
	var in store.Profile
	if !decode(w, r, &in) {
		return
	}
	in.UpdatedAt = nil
	errs := fieldErrors{}
	in.Skills = cleanList(errs, "skills", in.Skills, 100, 100)
	in.Locations = cleanList(errs, "locations", in.Locations, 20, 100)
	in.WorkModes = cleanList(errs, "workModes", in.WorkModes, 3, 10)
	for i, m := range in.WorkModes {
		in.WorkModes[i] = strings.ToLower(m)
		if !slices.Contains(workModes, in.WorkModes[i]) {
			errs["workModes"] = "each must be onsite, hybrid or remote"
		}
	}
	in.Resume = strings.TrimSpace(strings.ReplaceAll(in.Resume, "\r\n", "\n"))
	if utf8.RuneCountInString(in.Resume) > 30_000 {
		errs["resume"] = "is too long"
	}
	if in.ExperienceYears != nil && (*in.ExperienceYears < 0 || *in.ExperienceYears > 60) {
		errs["experienceYears"] = "must be between 0 and 60"
	}
	if in.MinPay != nil && *in.MinPay < 0 {
		errs["minPay"] = "must not be negative"
	}
	in.PayCurrency, in.PayPeriod = trimmed(in.PayCurrency), trimmed(in.PayPeriod)
	if in.PayCurrency != nil {
		upper := strings.ToUpper(*in.PayCurrency)
		in.PayCurrency = &upper
		if !currencyPattern.MatchString(upper) {
			errs["payCurrency"] = "must be a 3-letter code such as INR or USD"
		}
	}
	if in.PayPeriod != nil && !slices.Contains(payPeriods, *in.PayPeriod) {
		errs["payPeriod"] = "must be hour, month, year or total"
	}
	if in.MinPay != nil && (in.PayCurrency == nil || in.PayPeriod == nil) {
		errs["minPay"] = "needs a currency and a period"
	}
	if len(errs) > 0 {
		writeValidation(w, errs)
		return
	}
	p, err := s.store.SaveProfile(r.Context(), userID(r), in)
	if err != nil {
		serverError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// cleanList trims, drops empties and case-insensitive duplicates, and checks limits.
func cleanList(errs fieldErrors, field string, items []string, maxItems, maxLen int) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, s := range items {
		s = strings.TrimSpace(s)
		if s == "" || seen[strings.ToLower(s)] {
			continue
		}
		if utf8.RuneCountInString(s) > maxLen {
			errs[field] = "has an entry that is too long"
		}
		seen[strings.ToLower(s)] = true
		out = append(out, s)
	}
	if len(out) > maxItems {
		errs[field] = "has too many entries"
	}
	return out
}

// jobView is a job as the API returns it: the stored job plus its fit, which
// is computed on read so it follows profile changes. Fit is null until the
// posting has been parsed.
type jobView struct {
	store.Job
	Fit *fit.Result `json:"fit"`
}

func withFit(j store.Job, prof store.Profile) jobView {
	v := jobView{Job: j}
	if len(j.Parsed) == 0 {
		return v
	}
	var p agent.Posting
	if err := json.Unmarshal(j.Parsed, &p); err != nil {
		return v
	}
	r := fit.Score(p, prof)
	v.Fit = &r
	return v
}
