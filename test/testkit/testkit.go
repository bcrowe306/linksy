// Package testkit spins up an isolated Lynqr instance (SQLite + local storage)
// and provides helpers for authenticating users and calling the HTTP API.
package testkit

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/bcrowe306/linksy/internal/api"
	"github.com/bcrowe306/linksy/internal/config"
	"github.com/bcrowe306/linksy/internal/db"
	"github.com/bcrowe306/linksy/internal/models"
	"github.com/bcrowe306/linksy/internal/repository"
	"github.com/bcrowe306/linksy/internal/service"
	"github.com/bcrowe306/linksy/internal/storage"
)

const AppURL = "http://example.com"

type Kit struct {
	T    *testing.T
	App  *fiber.App
	Svc  *service.Service
	Repo *repository.Repository
	DB   *db.DB
}

// User is an authenticated test identity.
type User struct {
	ID     string
	Email  string
	Token  string
	TeamID string
}

func New(t *testing.T) *Kit {
	t.Helper()
	dir := t.TempDir()
	database := connect(t, dir)

	store, err := storage.NewLocalStorage(filepath.Join(dir, "storage"), AppURL)
	if err != nil {
		t.Fatalf("storage: %v", err)
	}
	repo := repository.New(database)
	svc := service.New(repo, store, AppURL)
	svc.SetIPSalt("test-salt")

	app := fiber.New()
	api.SetupRouter(app, svc, &config.Config{AppURL: AppURL})
	return &Kit{T: t, App: app, Svc: svc, Repo: repo, DB: database}
}

// connect opens SQLite by default, or an isolated PostgreSQL schema when
// LYNQR_TEST_POSTGRES_URL is set (CI runs the suite against both).
func connect(t *testing.T, dir string) *db.DB {
	t.Helper()
	pgURL := os.Getenv("LYNQR_TEST_POSTGRES_URL")
	if pgURL == "" {
		database, err := db.Connect("sqlite", filepath.Join(dir, "test.db"))
		if err != nil {
			t.Fatalf("connect: %v", err)
		}
		t.Cleanup(func() { database.Close() })
		return database
	}

	admin, err := sql.Open("postgres", pgURL)
	if err != nil {
		t.Fatalf("postgres: %v", err)
	}
	schema := fmt.Sprintf("t_%d_%d", time.Now().UnixNano(), rand.Intn(1_000_000))
	if _, err := admin.Exec("CREATE SCHEMA " + schema); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	sep := "?"
	if strings.Contains(pgURL, "?") {
		sep = "&"
	}
	database, err := db.Connect("postgres", pgURL+sep+"search_path="+schema)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	t.Cleanup(func() {
		database.Close()
		_, _ = admin.Exec("DROP SCHEMA " + schema + " CASCADE")
		admin.Close()
	})
	return database
}

// Login creates (or reuses) a user and returns a fresh session.
func (k *Kit) Login(email string) *User {
	k.T.Helper()
	id := "user_" + strings.NewReplacer("@", "_", ".", "_").Replace(email)
	token, sess, err := k.Svc.StartSession(&models.User{ID: id, Email: email, FirstName: strings.Split(email, "@")[0]}, "127.0.0.1", "testkit")
	if err != nil {
		k.T.Fatalf("start session: %v", err)
	}
	return &User{ID: id, Email: email, Token: token, TeamID: sess.ActiveTeamID}
}

// Actor returns a service actor for the user in the given team with their live role.
func (k *Kit) Actor(u *User, teamID string) service.Actor {
	k.T.Helper()
	tid, role, err := k.Svc.ResolveTeam(u.ID, teamID, "")
	if err != nil {
		k.T.Fatalf("resolve team: %v", err)
	}
	return service.Actor{UserID: u.ID, TeamID: tid, Role: role}
}

// Response is a decoded HTTP response.
type Response struct {
	Status int
	Header http.Header
	Body   []byte
}

func (r *Response) JSON(v any) error { return json.Unmarshal(r.Body, v) }

func (r *Response) Map() map[string]any {
	m := map[string]any{}
	_ = json.Unmarshal(r.Body, &m)
	return m
}

// Do issues a request. token may be a session token, API key, or "" for anonymous.
// Extra headers are passed as key/value pairs.
func (k *Kit) Do(method, path, token string, body any, headers ...string) *Response {
	k.T.Helper()
	var rdr io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, rdr)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	return k.Send(req)
}

func (k *Kit) Send(req *http.Request) *Response {
	k.T.Helper()
	resp, err := k.App.Test(req, fiber.TestConfig{Timeout: 0})
	if err != nil {
		k.T.Fatalf("%s %s: %v", req.Method, req.URL.Path, err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return &Response{Status: resp.StatusCode, Header: resp.Header, Body: b}
}

// Expect fails the test when the status code differs.
func (k *Kit) Expect(r *Response, status int, context string) {
	k.T.Helper()
	if r.Status != status {
		k.T.Fatalf("%s: expected %d, got %d: %s", context, status, r.Status, string(r.Body))
	}
}
