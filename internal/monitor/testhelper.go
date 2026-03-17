package monitor

import (
	"context"
	"testing"

	"github.com/alexpls/untils/internal/llm"
	"github.com/alexpls/untils/internal/models"
	"github.com/alexpls/untils/internal/notifications"
	"github.com/alexpls/untils/internal/testhelper"
	testfixtures "github.com/alexpls/untils/internal/testhelper/fixtures"
	"github.com/go-playground/validator/v10"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/stretchr/testify/require"
)

type testDeps struct {
	service  *Service
	handlers *Handlers
	fixtures testfixtures.Fixtures
}

func setupTestDeps(ctx context.Context, t *testing.T) testDeps {
	logger := testhelper.TestLogger(t)
	db := testhelper.TestTx(ctx, t)
	llm := llmWorkflowsStub{}
	river, err := river.NewClient(riverpgxv5.New(nil), &river.Config{})
	require.NoError(t, err)
	notificationSender := notificationSenderStub{}
	validator := validator.New(validator.WithRequiredStructEnabled())
	dbEvents := DBEventHandler{}
	queries := models.New()
	fixtures := testfixtures.New(ctx, t, db, queries)

	service := NewService(db, queries, &llm, river, logger, validator, notifications.Capabilities{
		EmailEnabled:    true,
		PushoverEnabled: true,
	}, &notificationSender, notifications.RenderConfig{
		BaseURL: "https://untils.example.com",
	})
	handlers := NewHandlers(service, &dbEvents, logger)

	return testDeps{
		service:  service,
		handlers: handlers,
		fixtures: fixtures,
	}
}

type llmWorkflowsStub struct{}

var _ llmWorkflowBuilder = &llmWorkflowsStub{}

func (s *llmWorkflowsStub) NewCheckWorkflow() llm.CheckWorkflowRunner {
	return &llm.CheckWorkflow{}
}

func (s *llmWorkflowsStub) NewTriageWorkflow() llm.TriageWorkflowRunner {
	return &llm.TriageWorkflow{}
}

type notificationSenderStub struct{}

var _ notifications.Sender = &notificationSenderStub{}

func (s *notificationSenderStub) Send(ctx context.Context, params notifications.SendParams) error {
	return nil
}
