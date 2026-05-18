package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/alexpls/untils/internal/apimessage"
	appdb "github.com/alexpls/untils/internal/db"
	"github.com/alexpls/untils/internal/models"
	"github.com/alexpls/untils/internal/monitor"
	"github.com/alexpls/untils/internal/notifications"
	"github.com/alexpls/untils/internal/reqcontext"
	"github.com/alexpls/untils/internal/testhelper"
	testfixtures "github.com/alexpls/untils/internal/testhelper/fixtures"
	"github.com/alexpls/untils/internal/validation"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenAuthentication(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	deps := setupTestDeps(ctx, t)

	created, err := deps.service.CreateToken(ctx, CreateTokenParams{
		UserID: deps.fixtures.User.ID,
		Name:   "test token",
	})
	require.NoError(t, err)
	require.NotEmpty(t, created.Key)
	require.Contains(t, created.Key, tokenPrefix)

	token, err := deps.service.AuthenticateToken(ctx, created.Key)
	require.NoError(t, err)
	assert.Equal(t, created.Token.ID, token.ID)
	assert.Equal(t, "test token", token.Name)

	// Wait for the async last_used_at write to complete before asserting on it,
	// and to avoid sharing the test transaction with the background goroutine.
	deps.service.WaitForBackgroundWrites()

	tokens, err := deps.service.ListTokens(ctx, deps.fixtures.User.ID)
	require.NoError(t, err)
	require.Len(t, tokens, 1)
	require.NotNil(t, tokens[0].LastUsedAt)

	_, err = deps.service.AuthenticateToken(ctx, created.Key+"nope")
	require.ErrorIs(t, err, ErrInvalidToken)
}

func TestCreateTokenAllowsDuplicateNamesPerUser(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	deps := setupTestDeps(ctx, t)

	first, err := deps.service.CreateToken(ctx, CreateTokenParams{
		UserID: deps.fixtures.User.ID,
		Name:   "deployments",
	})
	require.NoError(t, err)

	second, err := deps.service.CreateToken(ctx, CreateTokenParams{
		UserID: deps.fixtures.User.ID,
		Name:   "deployments",
	})
	require.NoError(t, err)
	require.NotEqual(t, first.Token.ID, second.Token.ID)
}

func TestCreateTokenNameMaxLength(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	deps := setupTestDeps(ctx, t)

	// 100 chars is allowed.
	atLimit := strings.Repeat("a", 100)
	created, err := deps.service.CreateToken(ctx, CreateTokenParams{
		UserID: deps.fixtures.User.ID,
		Name:   atLimit,
	})
	require.NoError(t, err)
	require.NotNil(t, created)
	assert.Equal(t, atLimit, created.Token.Name)

	// 101 chars fails with a validation error on the Name field.
	overLimit := strings.Repeat("a", 101)
	_, err = deps.service.CreateToken(ctx, CreateTokenParams{
		UserID: deps.fixtures.User.ID,
		Name:   overLimit,
	})
	require.Error(t, err)

	validationErrs := validation.MapValidationErrors(err)
	require.NotNil(t, validationErrs)
	require.Len(t, validationErrs, 1)
	assert.Equal(t, "Name", validationErrs[0].Field)
	assert.Equal(t, "This must be at most 100 characters", validationErrs[0].Message)
}

func TestDeleteTokenSuccess(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	deps := setupTestDeps(ctx, t)

	created, err := deps.service.CreateToken(ctx, CreateTokenParams{
		UserID: deps.fixtures.User.ID,
		Name:   "to delete",
	})
	require.NoError(t, err)

	err = deps.service.DeleteToken(ctx, deps.fixtures.User.ID, strconv.FormatInt(created.Token.ID, 10))
	require.NoError(t, err)

	tokens, err := deps.service.ListTokens(ctx, deps.fixtures.User.ID)
	require.NoError(t, err)
	assert.Empty(t, tokens)
}

func TestDeleteTokenNonExistentReturnsNotFound(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	deps := setupTestDeps(ctx, t)

	err := deps.service.DeleteToken(ctx, deps.fixtures.User.ID, "99999999")
	require.ErrorIs(t, err, ErrNotFound)

	err = deps.service.DeleteToken(ctx, deps.fixtures.User.ID, "not-a-number")
	require.ErrorIs(t, err, ErrNotFound)

	err = deps.service.DeleteToken(ctx, deps.fixtures.User.ID, "0")
	require.ErrorIs(t, err, ErrNotFound)

	err = deps.service.DeleteToken(ctx, deps.fixtures.User.ID, "-1")
	require.ErrorIs(t, err, ErrNotFound)
}

func TestDeleteTokenWrongUserReturnsNotFound(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	deps := setupTestDeps(ctx, t)
	otherUser := testfixtures.New(ctx, t, deps.db, deps.queries)

	created, err := deps.service.CreateToken(ctx, CreateTokenParams{
		UserID: deps.fixtures.User.ID,
		Name:   "owned by user",
	})
	require.NoError(t, err)

	err = deps.service.DeleteToken(ctx, otherUser.User.ID, strconv.FormatInt(created.Token.ID, 10))
	require.ErrorIs(t, err, ErrNotFound)

	// Token must still exist for the original owner.
	tokens, err := deps.service.ListTokens(ctx, deps.fixtures.User.ID)
	require.NoError(t, err)
	require.Len(t, tokens, 1)
	assert.Equal(t, created.Token.ID, tokens[0].ID)
}

func TestListLatestResultsHandler(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	deps := setupTestDeps(ctx, t)
	queries := deps.queries
	db := deps.db

	mon, err := queries.UpdateMonitorStatus(ctx, db, &models.UpdateMonitorStatusParams{
		Status: models.MonitorStatusActive,
		UserID: deps.fixtures.User.ID,
		ID:     deps.fixtures.Monitor.ID,
	})
	require.NoError(t, err)
	older := createResult(ctx, t, deps, mon.ID, "Older {{price}}")
	newer := createResult(ctx, t, deps, mon.ID, "Newer")

	paused, err := queries.CreateMonitor(ctx, db, &models.CreateMonitorParams{
		UserID:  deps.fixtures.User.ID,
		Subject: pgtype.Text{String: "Paused monitor", Valid: true},
	})
	require.NoError(t, err)
	paused, err = queries.UpdateMonitorStatus(ctx, db, &models.UpdateMonitorStatusParams{
		Status: models.MonitorStatusPaused,
		UserID: deps.fixtures.User.ID,
		ID:     paused.ID,
	})
	require.NoError(t, err)
	pausedResult := createResult(ctx, t, deps, paused.ID, "Paused")

	hidden := createResult(ctx, t, deps, mon.ID, "Hidden")
	require.NoError(t, queries.HideMonitorResult(ctx, db, hidden.ID))

	ready, err := queries.CreateMonitor(ctx, db, &models.CreateMonitorParams{
		UserID:  deps.fixtures.User.ID,
		Subject: pgtype.Text{String: "Ready monitor", Valid: true},
	})
	require.NoError(t, err)
	createResult(ctx, t, deps, ready.ID, "Ready")

	otherUser := testfixtures.New(ctx, t, db, queries)
	otherMon, err := queries.UpdateMonitorStatus(ctx, db, &models.UpdateMonitorStatusParams{
		Status: models.MonitorStatusActive,
		UserID: otherUser.User.ID,
		ID:     otherUser.Monitor.ID,
	})
	require.NoError(t, err)
	createResult(ctx, t, deps, otherMon.ID, "Other user")

	createdToken, err := deps.service.CreateToken(ctx, CreateTokenParams{
		UserID: deps.fixtures.User.ID,
		Name:   "results token",
	})
	require.NoError(t, err)

	var response struct {
		Error any `json:"error"`
		Data  struct {
			Type            string `json:"type"`
			ResultSummaries []struct {
				Type           string `json:"type"`
				ID             int64  `json:"id"`
				MonitorID      int64  `json:"monitor_id"`
				MonitorSubject string `json:"monitor_subject"`
				CreatedAt      string `json:"created_at"`
				Headline       string `json:"headline"`
				Fields         []struct {
					Name  string `json:"name"`
					Value string `json:"value"`
				} `json:"fields"`
			} `json:"result_summaries"`
		} `json:"data"`
	}
	deps.getAPIJSON(t, deps.handlers.ListLatestResults, "/api/results.list_latest", createdToken.Key, http.StatusOK, &response)
	require.Nil(t, response.Error)
	require.Equal(t, "result_summaries", response.Data.Type)
	require.Len(t, response.Data.ResultSummaries, 3)

	assert.Equal(t, "result_summary", response.Data.ResultSummaries[2].Type)
	assert.Equal(t, pausedResult.ID, response.Data.ResultSummaries[0].ID)
	assert.Equal(t, newer.ID, response.Data.ResultSummaries[1].ID)
	assert.Equal(t, older.ID, response.Data.ResultSummaries[2].ID)
	assert.Equal(t, "Older $12", response.Data.ResultSummaries[2].Headline)
	assert.Equal(t, "price", response.Data.ResultSummaries[2].Fields[0].Name)
	assert.Equal(t, "$12", response.Data.ResultSummaries[2].Fields[0].Value)
	assert.Equal(t, mon.ID, response.Data.ResultSummaries[2].MonitorID)
	assert.Equal(t, mon.Subject.String, response.Data.ResultSummaries[2].MonitorSubject)
	assert.NotEmpty(t, response.Data.ResultSummaries[2].CreatedAt)
}

func TestListLatestResultsHandlerLimitsResults(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	deps := setupTestDeps(ctx, t)

	mon, err := deps.queries.UpdateMonitorStatus(ctx, deps.db, &models.UpdateMonitorStatusParams{
		Status: models.MonitorStatusActive,
		UserID: deps.fixtures.User.ID,
		ID:     deps.fixtures.Monitor.ID,
	})
	require.NoError(t, err)

	for i := 0; i < latestResultsLimit+1; i++ {
		createResult(ctx, t, deps, mon.ID, "Result")
	}

	createdToken, err := deps.service.CreateToken(ctx, CreateTokenParams{
		UserID: deps.fixtures.User.ID,
		Name:   "results token",
	})
	require.NoError(t, err)

	var response struct {
		Data struct {
			ResultSummaries []struct {
				ID int64 `json:"id"`
			} `json:"result_summaries"`
		} `json:"data"`
	}
	deps.getAPIJSON(t, deps.handlers.ListLatestResults, "/api/results.list_latest", createdToken.Key, http.StatusOK, &response)
	require.Len(t, response.Data.ResultSummaries, latestResultsLimit)
}

func TestGetMonitorHandler(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	deps := setupTestDeps(ctx, t)

	createdToken, err := deps.service.CreateToken(ctx, CreateTokenParams{
		UserID: deps.fixtures.User.ID,
		Name:   "monitor token",
	})
	require.NoError(t, err)

	var response struct {
		Error any `json:"error"`
		Data  struct {
			Type    string `json:"type"`
			Monitor struct {
				Type      string `json:"type"`
				ID        int64  `json:"id"`
				CreatedAt string `json:"created_at"`
				Status    string `json:"status"`
				Subject   string `json:"subject"`
			} `json:"monitor"`
		} `json:"data"`
	}
	deps.getAPIJSON(t, deps.handlers.GetMonitor, "/api/monitor.get?monitor_id="+strconv.FormatInt(deps.fixtures.Monitor.ID, 10), createdToken.Key, http.StatusOK, &response)
	require.Nil(t, response.Error)
	require.Equal(t, "monitor", response.Data.Type)
	assert.Equal(t, "monitor", response.Data.Monitor.Type)
	assert.Equal(t, deps.fixtures.Monitor.ID, response.Data.Monitor.ID)
	assert.NotEmpty(t, response.Data.Monitor.CreatedAt)
	assert.Equal(t, string(deps.fixtures.Monitor.Status), response.Data.Monitor.Status)
	assert.Equal(t, deps.fixtures.Monitor.Subject.String, response.Data.Monitor.Subject)
}

func TestListResultsHandler(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	deps := setupTestDeps(ctx, t)

	mon, err := deps.queries.UpdateMonitorStatus(ctx, deps.db, &models.UpdateMonitorStatusParams{
		Status: models.MonitorStatusActive,
		UserID: deps.fixtures.User.ID,
		ID:     deps.fixtures.Monitor.ID,
	})
	require.NoError(t, err)
	older := createResult(ctx, t, deps, mon.ID, "Older {{price}}")
	newer := createResult(ctx, t, deps, mon.ID, "Newer")

	hidden := createResult(ctx, t, deps, mon.ID, "Hidden")
	require.NoError(t, deps.queries.HideMonitorResult(ctx, deps.db, hidden.ID))
	require.NoError(t, deps.queries.UpdateMonitorResultCorrection(ctx, deps.db, &models.UpdateMonitorResultCorrectionParams{
		ResultCorrection: pgtype.Text{String: "Needs correction", Valid: true},
		MonitorResultID:  hidden.ID,
	}))

	otherUser := testfixtures.New(ctx, t, deps.db, deps.queries)
	createResult(ctx, t, deps, otherUser.Monitor.ID, "Other user")

	createdToken, err := deps.service.CreateToken(ctx, CreateTokenParams{
		UserID: deps.fixtures.User.ID,
		Name:   "results token",
	})
	require.NoError(t, err)

	var response struct {
		Error any `json:"error"`
		Data  struct {
			Type    string `json:"type"`
			Results []struct {
				Type       string  `json:"type"`
				ID         int64   `json:"id"`
				Hidden     bool    `json:"hidden"`
				Correction *string `json:"correction"`
				Headline   string  `json:"headline"`
				Fields     []struct {
					Name  string `json:"name"`
					Value string `json:"value"`
				} `json:"fields"`
			} `json:"results"`
			Links struct {
				Next *string `json:"next"`
				Prev *string `json:"prev"`
			} `json:"links"`
		} `json:"data"`
	}
	deps.getAPIJSON(t, deps.handlers.ListResults, "/api/results.list?monitor_id="+strconv.FormatInt(mon.ID, 10), createdToken.Key, http.StatusOK, &response)
	require.Nil(t, response.Error)
	require.Equal(t, "results", response.Data.Type)
	require.Len(t, response.Data.Results, 3)
	assert.Equal(t, "result", response.Data.Results[0].Type)
	assert.Equal(t, hidden.ID, response.Data.Results[0].ID)
	assert.True(t, response.Data.Results[0].Hidden)
	require.NotNil(t, response.Data.Results[0].Correction)
	assert.Equal(t, "Needs correction", *response.Data.Results[0].Correction)
	assert.Equal(t, newer.ID, response.Data.Results[1].ID)
	assert.Equal(t, older.ID, response.Data.Results[2].ID)
	assert.Equal(t, "Older $12", response.Data.Results[2].Headline)
	assert.Equal(t, "price", response.Data.Results[2].Fields[0].Name)
	assert.Equal(t, "$12", response.Data.Results[2].Fields[0].Value)
	assert.Nil(t, response.Data.Links.Next)
	assert.Nil(t, response.Data.Links.Prev)
}

func TestListResultsHandlerPaginates(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	deps := setupTestDeps(ctx, t)

	mon, err := deps.queries.UpdateMonitorStatus(ctx, deps.db, &models.UpdateMonitorStatusParams{
		Status: models.MonitorStatusActive,
		UserID: deps.fixtures.User.ID,
		ID:     deps.fixtures.Monitor.ID,
	})
	require.NoError(t, err)
	older := createResult(ctx, t, deps, mon.ID, "Older")
	newer := createResult(ctx, t, deps, mon.ID, "Newer")
	newest := createResult(ctx, t, deps, mon.ID, "Newest")

	createdToken, err := deps.service.CreateToken(ctx, CreateTokenParams{
		UserID: deps.fixtures.User.ID,
		Name:   "results token",
	})
	require.NoError(t, err)

	type resultsPageResponse struct {
		Data struct {
			Results []struct {
				ID int64 `json:"id"`
			} `json:"results"`
			Links struct {
				Next *string `json:"next"`
				Prev *string `json:"prev"`
			} `json:"links"`
		} `json:"data"`
	}

	var response resultsPageResponse
	deps.getAPIJSON(t, deps.handlers.ListResults, "/api/results.list?monitor_id="+strconv.FormatInt(mon.ID, 10)+"&per_page=2", createdToken.Key, http.StatusOK, &response)
	require.Len(t, response.Data.Results, 2)
	assert.Equal(t, newest.ID, response.Data.Results[0].ID)
	assert.Equal(t, newer.ID, response.Data.Results[1].ID)
	require.NotNil(t, response.Data.Links.Next)
	assert.Equal(t, testBaseURL+"/api/results.list?monitor_id="+strconv.FormatInt(mon.ID, 10)+"&page=2&per_page=2", *response.Data.Links.Next)
	assert.Nil(t, response.Data.Links.Prev)

	response = resultsPageResponse{}
	deps.getAPIJSON(t, deps.handlers.ListResults, "/api/results.list?monitor_id="+strconv.FormatInt(mon.ID, 10)+"&page=2&per_page=2", createdToken.Key, http.StatusOK, &response)
	require.Len(t, response.Data.Results, 1)
	assert.Equal(t, older.ID, response.Data.Results[0].ID)
	assert.Nil(t, response.Data.Links.Next)
	require.NotNil(t, response.Data.Links.Prev)
	assert.Equal(t, testBaseURL+"/api/results.list?monitor_id="+strconv.FormatInt(mon.ID, 10)+"&page=1&per_page=2", *response.Data.Links.Prev)
}

func TestListResultsHandlerHandlesLargePageOffset(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	deps := setupTestDeps(ctx, t)

	mon, err := deps.queries.UpdateMonitorStatus(ctx, deps.db, &models.UpdateMonitorStatusParams{
		Status: models.MonitorStatusActive,
		UserID: deps.fixtures.User.ID,
		ID:     deps.fixtures.Monitor.ID,
	})
	require.NoError(t, err)
	createResult(ctx, t, deps, mon.ID, "Result")

	createdToken, err := deps.service.CreateToken(ctx, CreateTokenParams{
		UserID: deps.fixtures.User.ID,
		Name:   "large page token",
	})
	require.NoError(t, err)

	res := deps.getAPI(t, deps.handlers.ListResults, "/api/results.list?monitor_id="+strconv.FormatInt(mon.ID, 10)+"&page=2147483648&per_page=1", createdToken.Key)
	require.Equal(t, http.StatusNotFound, res.Code)
	assertErrorCode(t, res, "not_found")
}

func TestAPIResourceOwnershipAndQueryErrors(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	deps := setupTestDeps(ctx, t)
	otherUser := testfixtures.New(ctx, t, deps.db, deps.queries)

	createdToken, err := deps.service.CreateToken(ctx, CreateTokenParams{
		UserID: deps.fixtures.User.ID,
		Name:   "monitor token",
	})
	require.NoError(t, err)

	res := deps.getAPI(t, deps.handlers.GetMonitor, "/api/monitor.get?monitor_id="+strconv.FormatInt(otherUser.Monitor.ID, 10), createdToken.Key)
	require.Equal(t, http.StatusNotFound, res.Code)
	assertErrorCode(t, res, "not_found")

	res = deps.getAPI(t, deps.handlers.ListResults, "/api/results.list?monitor_id="+strconv.FormatInt(otherUser.Monitor.ID, 10), createdToken.Key)
	require.Equal(t, http.StatusNotFound, res.Code)
	assertErrorCode(t, res, "not_found")

	res = deps.getAPI(t, deps.handlers.GetMonitor, "/api/monitor.get", createdToken.Key)
	require.Equal(t, http.StatusBadRequest, res.Code)
	assertErrorCode(t, res, "bad_request")

	res = deps.getAPI(t, deps.handlers.ListResults, "/api/results.list?monitor_id=nope", createdToken.Key)
	require.Equal(t, http.StatusBadRequest, res.Code)
	assertErrorCode(t, res, "bad_request")
}

func TestAPIErrors(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	deps := setupTestDeps(ctx, t)

	res := deps.getAPI(t, deps.handlers.ListLatestResults, "/api/results.list_latest", "")
	require.Equal(t, http.StatusUnauthorized, res.Code)
	assertErrorCode(t, res, "unauthorized")

	createdToken, err := deps.service.CreateToken(ctx, CreateTokenParams{
		UserID: deps.fixtures.User.ID,
		Name:   "not found token",
	})
	require.NoError(t, err)

	res = deps.requestAPI(t, deps.handlers.ListLatestResults, apiRequest(http.MethodPut, "/api/results.list_latest", createdToken.Key))
	require.Equal(t, http.StatusMethodNotAllowed, res.Code)
	require.Equal(t, "GET", res.Header().Get("Allow"))
	assertErrorCode(t, res, "method_not_allowed")

	res = deps.requestAPI(t, deps.handlers.ListLatestResults, apiRequest(http.MethodPost, "/api/results.list_latest", createdToken.Key))
	require.Equal(t, http.StatusMethodNotAllowed, res.Code)
	require.Equal(t, "GET", res.Header().Get("Allow"))
	assertErrorCode(t, res, "method_not_allowed")

	res = deps.getAPI(t, deps.handlers.NotFound, "/api/nope", createdToken.Key)
	require.Equal(t, http.StatusNotFound, res.Code)
	assertErrorCode(t, res, "not_found")
}

type testDeps struct {
	db       appdb.DB
	queries  *models.Queries
	service  *Service
	handlers *Handlers
	fixtures testfixtures.Fixtures
}

const testBaseURL = "https://untils.test"

func (d testDeps) getAPI(t *testing.T, handler http.HandlerFunc, target, token string) *httptest.ResponseRecorder {
	t.Helper()

	return d.requestAPI(t, handler, apiRequest(http.MethodGet, target, token))
}

func (d testDeps) getAPIJSON(t *testing.T, handler http.HandlerFunc, target, token string, status int, response any) *httptest.ResponseRecorder {
	t.Helper()

	return d.requestAPIJSON(t, handler, apiRequest(http.MethodGet, target, token), status, response)
}

func (d testDeps) requestAPI(t *testing.T, handler http.HandlerFunc, request testhelper.HTTPRequest) *httptest.ResponseRecorder {
	t.Helper()

	return testhelper.ServeHTTP(t, d.apiHandler(handler), request)
}

func (d testDeps) requestAPIJSON(t *testing.T, handler http.HandlerFunc, request testhelper.HTTPRequest, status int, response any) *httptest.ResponseRecorder {
	t.Helper()

	return testhelper.ServeJSON(t, d.apiHandler(handler), request, status, response)
}

func (d testDeps) apiHandler(handler http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		req = req.WithContext(reqcontext.ContextWithBaseURL(req.Context(), testBaseURL))

		next := http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			// AuthenticateToken kicks off an async last_used_at update that uses the
			// test transaction. Wait before the handler performs more queries on the
			// shared tx, then again after the request for deterministic cleanup.
			d.service.WaitForBackgroundWrites()
			d.handlers.RequireMethodGet(handler).ServeHTTP(res, req)
		})

		d.handlers.RequireToken(next).ServeHTTP(res, req)
		d.service.WaitForBackgroundWrites()
	})
}

func apiRequest(method, target, token string) testhelper.HTTPRequest {
	header := http.Header{}
	if token != "" {
		header.Set("Authorization", "Bearer "+token)
	}

	return testhelper.HTTPRequest{
		Method: method,
		Target: target,
		Header: header,
	}
}

func setupTestDeps(ctx context.Context, t *testing.T) testDeps {
	t.Helper()

	db := testhelper.TestTx(ctx, t)
	queries := models.New()
	fixtures := testfixtures.New(ctx, t, db, queries)
	validate := validator.New(validator.WithRequiredStructEnabled())
	monitorService := monitor.NewService(db, queries, nil, nil, testhelper.TestLogger(t), validate, notifications.Capabilities{}, nil, notifications.RenderConfig{})
	service := NewService(db, queries, validate, testhelper.TestLogger(t))
	handlers := NewHandlers(service, monitorService, testhelper.TestLogger(t))

	// Ensure any background DB writes (e.g. async last_used_at updates) finish
	// before the test transaction is rolled back in its own cleanup.
	t.Cleanup(service.WaitForBackgroundWrites)

	return testDeps{
		db:       db,
		queries:  queries,
		service:  service,
		handlers: handlers,
		fixtures: fixtures,
	}
}

func createResult(ctx context.Context, t *testing.T, deps testDeps, monitorID int64, headline string) *models.MonitorResult {
	t.Helper()

	result, err := deps.queries.CreateMonitorResult(ctx, deps.db, &models.CreateMonitorResultParams{
		MonitorID: monitorID,
		Headline:  headline,
		Subtitle:  "",
		Data: models.MonitorUpdateData{
			Fields: models.MonitorUpdateFields{
				{
					MonitorSchemaField: models.MonitorSchemaField{
						Type: models.MonitorSchemaFieldTypeText,
						Name: "price",
					},
					Value: "$12",
				},
			},
		},
		Citations: &models.Citations{},
	})
	require.NoError(t, err)
	return result
}

func assertErrorCode(t *testing.T, res *httptest.ResponseRecorder, code string) {
	t.Helper()

	var response apimessage.ErrorResponse
	body := res.Body.Bytes()
	var raw map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(body, &raw))
	require.Contains(t, raw, "data")
	assert.Equal(t, "null", string(raw["data"]))
	require.NoError(t, json.Unmarshal(body, &response))
	assert.Equal(t, code, response.Error.Code)
}
