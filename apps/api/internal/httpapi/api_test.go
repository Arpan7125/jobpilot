package httpapi

// These tests run the real router against a real Postgres. Point
// TEST_DATABASE_URL at a database you don't mind being wiped; without it
// the tests are skipped.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Arpan7125/jobpilot/apps/api/internal/agent"
	"github.com/Arpan7125/jobpilot/apps/api/internal/auth"
	"github.com/Arpan7125/jobpilot/apps/api/internal/db"
	"github.com/Arpan7125/jobpilot/apps/api/internal/store"
)

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		fmt.Println("TEST_DATABASE_URL not set, skipping API tests")
		os.Exit(0)
	}
	ctx := context.Background()
	pool, err := db.Open(ctx, url)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if _, err := pool.Exec(ctx, `DROP SCHEMA public CASCADE; CREATE SCHEMA public`); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if err := db.Migrate(ctx, pool); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	testPool = pool
	code := m.Run()
	pool.Close()
	os.Exit(code)
}

type client struct {
	t     *testing.T
	h     http.Handler
	token string
}

func newClient(t *testing.T) *client {
	return newClientWithParser(t, nil)
}

func newClientWithParser(t *testing.T, p agent.Parser) *client {
	tokens, err := auth.NewTokens("test-secret-that-is-at-least-32-characters")
	if err != nil {
		t.Fatal(err)
	}
	return &client{t: t, h: New(store.New(testPool), tokens, []string{"http://localhost:3000"}, p).Handler()}
}

// do sends a request and decodes a JSON response into out (if non-nil).
func (c *client) do(method, path string, body any, out any) int {
	c.t.Helper()
	var buf bytes.Buffer
	if s, ok := body.(string); ok {
		buf.WriteString(s)
	} else if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	rec := httptest.NewRecorder()
	c.h.ServeHTTP(rec, req)
	if out != nil {
		if err := json.Unmarshal(rec.Body.Bytes(), out); err != nil {
			c.t.Fatalf("%s %s: decode %q: %v", method, path, rec.Body.String(), err)
		}
	}
	return rec.Code
}

var emailSeq atomic.Int64

// signedIn returns a client logged in as a brand-new user.
func signedIn(t *testing.T) *client {
	c := newClient(t)
	var res authResponse
	email := fmt.Sprintf("user%d@example.com", emailSeq.Add(1))
	if code := c.do("POST", "/auth/signup", map[string]string{"email": email, "password": "password123"}, &res); code != 201 {
		t.Fatalf("signup = %d", code)
	}
	c.token = res.Token
	return c
}

func validJob() map[string]any {
	return map[string]any{
		"company": "Acme", "title": "Backend Intern", "description": "Go, Postgres, Docker",
		"location": "Bengaluru", "url": "https://acme.example/jobs/1", "platform": "LinkedIn",
		"salaryMin": 20000, "salaryMax": 30000, "currency": "inr",
	}
}

func TestHealth(t *testing.T) {
	var body map[string]string
	if code := newClient(t).do("GET", "/health", nil, &body); code != 200 || body["db"] != "ok" {
		t.Fatalf("health = %d %v", code, body)
	}
}

func TestSignupAndLogin(t *testing.T) {
	c := newClient(t)
	creds := map[string]string{"email": "  Dana@Example.com ", "password": "password123", "name": "Dana"}

	var res authResponse
	if code := c.do("POST", "/auth/signup", creds, &res); code != 201 {
		t.Fatalf("signup = %d", code)
	}
	if res.User.Email != "dana@example.com" || res.Token == "" {
		t.Fatalf("signup response = %+v", res)
	}
	if code := c.do("POST", "/auth/signup", creds, nil); code != 409 {
		t.Errorf("duplicate signup = %d, want 409", code)
	}

	var login authResponse
	if code := c.do("POST", "/auth/login", map[string]string{"email": "DANA@example.com", "password": "password123"}, &login); code != 200 {
		t.Fatalf("login = %d", code)
	}
	c.token = login.Token
	var me store.User
	if code := c.do("GET", "/me", nil, &me); code != 200 || me.ID != res.User.ID {
		t.Fatalf("me = %d %+v", code, me)
	}

	for name, body := range map[string]map[string]string{
		"wrong password": {"email": "dana@example.com", "password": "wrong-password"},
		"unknown email":  {"email": "nobody@example.com", "password": "password123"},
	} {
		var e map[string]string
		if code := c.do("POST", "/auth/login", body, &e); code != 401 || e["error"] != "wrong email or password" {
			t.Errorf("%s: login = %d %v", name, code, e)
		}
	}
}

func TestSignupValidation(t *testing.T) {
	var e struct{ Fields map[string]string }
	code := newClient(t).do("POST", "/auth/signup", map[string]string{"email": "not-an-email", "password": "short"}, &e)
	if code != 422 || e.Fields["email"] == "" || e.Fields["password"] == "" {
		t.Fatalf("signup = %d %+v", code, e)
	}
}

func TestRoutesNeedAToken(t *testing.T) {
	c := newClient(t)
	for _, route := range []string{"GET /me", "GET /jobs", "POST /jobs",
		"GET /jobs/00000000-0000-0000-0000-000000000000",
		"PUT /jobs/00000000-0000-0000-0000-000000000000/status"} {
		method, path, _ := strings.Cut(route, " ")
		if code := c.do(method, path, nil, nil); code != 401 {
			t.Errorf("%s without token = %d, want 401", route, code)
		}
	}
	c.token = "not-a-jwt"
	if code := c.do("GET", "/jobs", nil, nil); code != 401 {
		t.Errorf("bad token = %d, want 401", code)
	}
}

func TestJobLifecycle(t *testing.T) {
	c := signedIn(t)

	var job store.Job
	if code := c.do("POST", "/jobs", validJob(), &job); code != 201 {
		t.Fatalf("create = %d", code)
	}
	if job.Status != "saved" || *job.Currency != "INR" || job.AppliedAt != nil {
		t.Fatalf("created job = %+v", job)
	}

	edit := validJob()
	edit["title"] = "Backend Engineer Intern"
	edit["notes"] = "Referred by Priya"
	delete(edit, "location")
	if code := c.do("PUT", "/jobs/"+job.ID, edit, &job); code != 200 {
		t.Fatalf("update = %d", code)
	}
	if job.Title != "Backend Engineer Intern" || job.Location != nil || *job.Notes != "Referred by Priya" {
		t.Fatalf("updated job = %+v", job)
	}

	if code := c.do("PUT", "/jobs/"+job.ID+"/status", map[string]string{"status": "applied"}, &job); code != 200 {
		t.Fatalf("status applied = %d", code)
	}
	if job.Status != "applied" || job.AppliedAt == nil {
		t.Fatalf("after applied = %+v", job)
	}
	appliedAt := *job.AppliedAt
	if code := c.do("PUT", "/jobs/"+job.ID+"/status", map[string]string{"status": "applied"}, nil); code != 409 {
		t.Errorf("same status = %d, want 409", code)
	}
	c.do("PUT", "/jobs/"+job.ID+"/status", map[string]string{"status": "interview", "note": "Round 1 on Monday"}, &job)
	if !job.AppliedAt.Equal(appliedAt) {
		t.Errorf("appliedAt moved from %v to %v", appliedAt, job.AppliedAt)
	}

	var history []store.StatusChange
	if code := c.do("GET", "/jobs/"+job.ID+"/history", nil, &history); code != 200 {
		t.Fatalf("history = %d", code)
	}
	var got []string
	for _, h := range history {
		from := "-"
		if h.FromStatus != nil {
			from = *h.FromStatus
		}
		got = append(got, from+">"+h.ToStatus)
	}
	if want := "->saved,saved>applied,applied>interview"; strings.Join(got, ",") != want {
		t.Errorf("history = %v, want %s", got, want)
	}

	var list []store.Job
	c.do("GET", "/jobs?status=interview", nil, &list)
	if len(list) != 1 || list[0].ID != job.ID {
		t.Errorf("filtered list = %+v", list)
	}
	c.do("GET", "/jobs?status=offer", nil, &list)
	if len(list) != 0 {
		t.Errorf("offer list = %+v", list)
	}

	if code := c.do("DELETE", "/jobs/"+job.ID, nil, nil); code != 204 {
		t.Fatalf("delete = %d", code)
	}
	if code := c.do("GET", "/jobs/"+job.ID, nil, nil); code != 404 {
		t.Errorf("get after delete = %d", code)
	}
}

func TestJobValidation(t *testing.T) {
	c := signedIn(t)
	bad := map[string]any{
		"company": "  ", "title": "", "description": "x",
		"url": "javascript:alert(1)", "salaryMin": 50, "salaryMax": 10, "currency": "rupees",
	}
	var e struct{ Fields map[string]string }
	if code := c.do("POST", "/jobs", bad, &e); code != 422 {
		t.Fatalf("create = %d", code)
	}
	for _, f := range []string{"company", "title", "url", "salaryMax", "currency"} {
		if e.Fields[f] == "" {
			t.Errorf("no error for %s: %v", f, e.Fields)
		}
	}

	if code := c.do("POST", "/jobs", `{"company": "Acme"`, nil); code != 400 {
		t.Errorf("broken JSON = %d, want 400", code)
	}
	if code := c.do("POST", "/jobs", `{"company":"A","title":"B","description":"C","userId":"x"}`, nil); code != 400 {
		t.Errorf("unknown field = %d, want 400", code)
	}

	var job store.Job
	c.do("POST", "/jobs", validJob(), &job)
	if code := c.do("PUT", "/jobs/"+job.ID+"/status", map[string]string{"status": "hired"}, nil); code != 422 {
		t.Errorf("bad status = %d, want 422", code)
	}
	if code := c.do("GET", "/jobs/not-a-uuid", nil, nil); code != 404 {
		t.Errorf("malformed id = %d, want 404", code)
	}
}

func TestUsersCannotSeeEachOthersJobs(t *testing.T) {
	alice, bob := signedIn(t), signedIn(t)
	var job store.Job
	alice.do("POST", "/jobs", validJob(), &job)

	for _, route := range []string{"GET /jobs/" + job.ID, "GET /jobs/" + job.ID + "/history", "DELETE /jobs/" + job.ID} {
		method, path, _ := strings.Cut(route, " ")
		if code := bob.do(method, path, nil, nil); code != 404 {
			t.Errorf("bob %s = %d, want 404", route, code)
		}
	}
	if code := bob.do("PUT", "/jobs/"+job.ID, validJob(), nil); code != 404 {
		t.Errorf("bob update = %d, want 404", code)
	}
	if code := bob.do("PUT", "/jobs/"+job.ID+"/status", map[string]string{"status": "rejected"}, nil); code != 404 {
		t.Errorf("bob status = %d, want 404", code)
	}
	var list []store.Job
	bob.do("GET", "/jobs", nil, &list)
	if len(list) != 0 {
		t.Errorf("bob sees %d jobs", len(list))
	}
	if code := alice.do("GET", "/jobs/"+job.ID, nil, &job); code != 200 || job.Status != "saved" {
		t.Errorf("alice's job changed: %d %+v", code, job)
	}
}

// backdate pretends the job's application entered its status `days` days ago.
func backdate(t *testing.T, jobID string, days int) {
	t.Helper()
	_, err := testPool.Exec(context.Background(),
		`UPDATE applications SET status_changed_at = now() - make_interval(days => $2) WHERE job_id = $1`, jobID, days)
	if err != nil {
		t.Fatal(err)
	}
}

func TestFollowUpReminders(t *testing.T) {
	ctx := context.Background()
	st := store.New(testPool)
	c, other := signedIn(t), signedIn(t)

	var stale, fresh, saved store.Job
	c.do("POST", "/jobs", validJob(), &stale)
	c.do("POST", "/jobs", validJob(), &fresh)
	c.do("POST", "/jobs", validJob(), &saved)
	c.do("PUT", "/jobs/"+stale.ID+"/status", map[string]string{"status": "applied"}, nil)
	c.do("PUT", "/jobs/"+fresh.ID+"/status", map[string]string{"status": "applied"}, nil)
	backdate(t, stale.ID, 8)
	backdate(t, fresh.ID, 6)
	backdate(t, saved.ID, 30) // old but never applied

	if _, err := st.FlagStaleApplications(ctx, store.FollowUpAfter); err != nil {
		t.Fatal(err)
	}
	// Running again must not add duplicates.
	if _, err := st.FlagStaleApplications(ctx, store.FollowUpAfter); err != nil {
		t.Fatal(err)
	}

	var reminders []store.Reminder
	c.do("GET", "/reminders", nil, &reminders)
	if len(reminders) != 1 || reminders[0].JobID != stale.ID {
		t.Fatalf("reminders = %+v, want one for the stale job", reminders)
	}
	var job store.Job
	c.do("GET", "/jobs/"+stale.ID, nil, &job)
	if !job.FollowUpDue {
		t.Error("stale job not marked followUpDue")
	}

	var others []store.Reminder
	other.do("GET", "/reminders", nil, &others)
	if len(others) != 0 {
		t.Errorf("other user sees %d reminders", len(others))
	}
	if code := other.do("POST", "/reminders/"+reminders[0].ID+"/dismiss", nil, nil); code != 404 {
		t.Errorf("other user dismiss = %d, want 404", code)
	}

	if code := c.do("POST", "/reminders/"+reminders[0].ID+"/dismiss", nil, nil); code != 204 {
		t.Fatalf("dismiss = %d", code)
	}
	st.FlagStaleApplications(ctx, store.FollowUpAfter)
	c.do("GET", "/reminders", nil, &reminders)
	if len(reminders) != 0 {
		t.Errorf("dismissed reminder came back: %+v", reminders)
	}

	// A status change closes the open reminder.
	backdate(t, fresh.ID, 9)
	st.FlagStaleApplications(ctx, store.FollowUpAfter)
	c.do("GET", "/reminders", nil, &reminders)
	if len(reminders) != 1 || reminders[0].JobID != fresh.ID {
		t.Fatalf("reminders = %+v, want one for the second job", reminders)
	}
	c.do("PUT", "/jobs/"+fresh.ID+"/status", map[string]string{"status": "replied"}, nil)
	c.do("GET", "/reminders", nil, &reminders)
	if len(reminders) != 0 {
		t.Errorf("reply did not close the reminder: %+v", reminders)
	}
}
