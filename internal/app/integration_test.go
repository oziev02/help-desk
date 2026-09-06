//go:build integration

package app_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/oziev02/help-desk/internal/app"
	"github.com/oziev02/help-desk/internal/config"
)

func setupTestApp(t *testing.T) *httptest.Server {
	t.Helper()
	cfg, err := config.Load()
	require.NoError(t, err)
	cfg.SeedDemo = true
	cfg.AppEnv = "dev"
	if cfg.JWTSecret == "" {
		cfg.JWTSecret = "dev-secret-change-me"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	application, err := app.New(ctx, cfg, app.NewLogger())
	require.NoError(t, err)
	t.Cleanup(application.Close)

	return httptest.NewServer(application.Handler())
}

func login(t *testing.T, srv *httptest.Server, email, password string) string {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"email": email, "password": password})
	resp, err := http.Post(srv.URL+"/api/v1/auth/login", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var out struct {
		AccessToken string `json:"access_token"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
	return out.AccessToken
}

func TestTicketFlow(t *testing.T) {
	srv := setupTestApp(t)
	defer srv.Close()

	userToken := login(t, srv, "user@helpdesk.local", "user123")
	dispatcherToken := login(t, srv, "dispatcher@helpdesk.local", "dispatcher123")
	executorToken := login(t, srv, "executor@helpdesk.local", "executor123")

	createBody, _ := json.Marshal(map[string]string{
		"title":       "Broken AC",
		"description": "Room 101 AC not working",
	})
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/tickets", bytes.NewReader(createBody))
	req.Header.Set("Authorization", "Bearer "+userToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var ticket struct {
		ID    string     `json:"id"`
		DueAt *time.Time `json:"due_at"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&ticket))
	require.NotNil(t, ticket.DueAt)

	assignBody, _ := json.Marshal(map[string]string{"assignee_id": fetchExecutorID(t, srv)})
	req, _ = http.NewRequest(http.MethodPost, srv.URL+"/api/v1/tickets/"+ticket.ID+"/assign", bytes.NewReader(assignBody))
	req.Header.Set("Authorization", "Bearer "+dispatcherToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	transition := func(token, status string) {
		t.Helper()
		b, _ := json.Marshal(map[string]string{"status": status})
		r, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/tickets/"+ticket.ID+"/transitions", bytes.NewReader(b))
		r.Header.Set("Authorization", "Bearer "+token)
		r.Header.Set("Content-Type", "application/json")
		res, e := http.DefaultClient.Do(r)
		require.NoError(t, e)
		defer res.Body.Close()
		require.Equal(t, http.StatusOK, res.StatusCode)
	}

	transition(executorToken, "in_progress")
	transition(executorToken, "resolved")

	completeBody, _ := json.Marshal(map[string]any{"rating": 5, "comment": "всё работает"})
	completeReq, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/tickets/"+ticket.ID+"/complete", bytes.NewReader(completeBody))
	completeReq.Header.Set("Authorization", "Bearer "+userToken)
	completeReq.Header.Set("Content-Type", "application/json")
	completeResp, err := http.DefaultClient.Do(completeReq)
	require.NoError(t, err)
	completeResp.Body.Close()
	require.Equal(t, http.StatusOK, completeResp.StatusCode)

	reopenReq, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/tickets/"+ticket.ID+"/reopen", nil)
	reopenReq.Header.Set("Authorization", "Bearer "+userToken)
	reopenResp, err := http.DefaultClient.Do(reopenReq)
	require.NoError(t, err)
	reopenResp.Body.Close()
	require.Equal(t, http.StatusOK, reopenResp.StatusCode)
}

func TestUserCannotAssign(t *testing.T) {
	srv := setupTestApp(t)
	defer srv.Close()

	userToken := login(t, srv, "user@helpdesk.local", "user123")
	createBody, _ := json.Marshal(map[string]string{"title": "Test", "description": "d"})
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/tickets", bytes.NewReader(createBody))
	req.Header.Set("Authorization", "Bearer "+userToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	var ticket struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&ticket))
	resp.Body.Close()

	assignBody, _ := json.Marshal(map[string]string{"assignee_id": "00000000-0000-0000-0000-000000000001"})
	req, _ = http.NewRequest(http.MethodPost, srv.URL+"/api/v1/tickets/"+ticket.ID+"/assign", bytes.NewReader(assignBody))
	req.Header.Set("Authorization", "Bearer "+userToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestExtrasRBACAndRooms(t *testing.T) {
	srv := setupTestApp(t)
	defer srv.Close()

	userToken := login(t, srv, "user@helpdesk.local", "user123")
	dispatcherToken := login(t, srv, "dispatcher@helpdesk.local", "dispatcher123")
	managerToken := login(t, srv, "manager@helpdesk.local", "manager123")
	adminToken := login(t, srv, "admin@helpdesk.local", "admin123")

	create := func(token, title string) string {
		t.Helper()
		body, _ := json.Marshal(map[string]string{"title": title, "description": "d"})
		req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/tickets", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		var ticket struct {
			ID string `json:"id"`
		}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&ticket))
		return ticket.ID
	}

	id1 := create(userToken, "Ticket A")
	id2 := create(userToken, "Ticket B")

	linkBody, _ := json.Marshal(map[string]string{"linked_ticket_id": id2})
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/tickets/"+id1+"/links", bytes.NewReader(linkBody))
	req.Header.Set("Authorization", "Bearer "+userToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusForbidden, resp.StatusCode)

	req, _ = http.NewRequest(http.MethodPost, srv.URL+"/api/v1/tickets/"+id1+"/links", bytes.NewReader(linkBody))
	req.Header.Set("Authorization", "Bearer "+dispatcherToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	req, _ = http.NewRequest(http.MethodGet, srv.URL+"/api/v1/reports/tickets", nil)
	req.Header.Set("Authorization", "Bearer "+userToken)
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusForbidden, resp.StatusCode)

	req, _ = http.NewRequest(http.MethodGet, srv.URL+"/api/v1/reports/tickets", nil)
	req.Header.Set("Authorization", "Bearer "+managerToken)
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	req, _ = http.NewRequest(http.MethodGet, srv.URL+"/api/v1/rooms", nil)
	req.Header.Set("Authorization", "Bearer "+userToken)
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var rooms []map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&rooms))
	require.NotEmpty(t, rooms)

	catBody, _ := json.Marshal(map[string]string{"name": "TempCat-" + time.Now().Format("150405")})
	req, _ = http.NewRequest(http.MethodPost, srv.URL+"/api/v1/admin/categories", bytes.NewReader(catBody))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	var cat struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&cat))
	resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	req, _ = http.NewRequest(http.MethodPatch, srv.URL+"/api/v1/admin/categories/"+cat.ID, nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var deactivated struct {
		IsActive bool `json:"is_active"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&deactivated))
	require.False(t, deactivated.IsActive)
}

func TestTicketIDORAndAssignedTransition(t *testing.T) {
	srv := setupTestApp(t)
	defer srv.Close()

	regA, _ := json.Marshal(map[string]string{
		"email": "alice-idor@helpdesk.local", "password": "password1", "full_name": "Alice",
	})
	regB, _ := json.Marshal(map[string]string{
		"email": "bob-idor@helpdesk.local", "password": "password1", "full_name": "Bob",
	})
	resp, err := http.Post(srv.URL+"/api/v1/auth/register", "application/json", bytes.NewReader(regA))
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var authA struct {
		AccessToken string `json:"access_token"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&authA))
	resp.Body.Close()

	resp, err = http.Post(srv.URL+"/api/v1/auth/register", "application/json", bytes.NewReader(regB))
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var authB struct {
		AccessToken string `json:"access_token"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&authB))
	resp.Body.Close()

	createBody, _ := json.Marshal(map[string]string{"title": "Alice ticket", "description": "private"})
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/tickets", bytes.NewReader(createBody))
	req.Header.Set("Authorization", "Bearer "+authA.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	var ticket struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&ticket))
	resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	req, _ = http.NewRequest(http.MethodGet, srv.URL+"/api/v1/tickets/"+ticket.ID, nil)
	req.Header.Set("Authorization", "Bearer "+authB.AccessToken)
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusNotFound, resp.StatusCode)

	commentBody, _ := json.Marshal(map[string]string{"body": "nosy"})
	req, _ = http.NewRequest(http.MethodPost, srv.URL+"/api/v1/tickets/"+ticket.ID+"/comments", bytes.NewReader(commentBody))
	req.Header.Set("Authorization", "Bearer "+authB.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusNotFound, resp.StatusCode)

	dispatcherToken := login(t, srv, "dispatcher@helpdesk.local", "dispatcher123")
	trBody, _ := json.Marshal(map[string]string{"status": "assigned"})
	req, _ = http.NewRequest(http.MethodPost, srv.URL+"/api/v1/tickets/"+ticket.ID+"/transitions", bytes.NewReader(trBody))
	req.Header.Set("Authorization", "Bearer "+dispatcherToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func fetchExecutorID(t *testing.T, srv *httptest.Server) string {
	t.Helper()
	// login as executor to get id from profile via register path - use known seed user via login response
	body, _ := json.Marshal(map[string]string{"email": "executor@helpdesk.local", "password": "executor123"})
	resp, err := http.Post(srv.URL+"/api/v1/auth/login", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	defer resp.Body.Close()
	var out struct {
		User struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
	return out.User.ID
}
