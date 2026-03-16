package monitor

import (
	"context"
	"testing"

	"github.com/alexpls/untils/internal/llm"
	"github.com/alexpls/untils/internal/models"
	"github.com/stretchr/testify/require"
)

func TestPerformMonitorCheckSendsNotification(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	deps := setupTestDeps(ctx, t)

	_, err := deps.service.queries.UpdateMonitorStatus(ctx, deps.service.db, &models.UpdateMonitorStatusParams{
		ID:     deps.fixtures.Monitor.ID,
		UserID: deps.fixtures.User.ID,
		Status: models.MonitorStatusActive,
	})
	require.NoError(t, err)

	_, err = deps.service.queries.CreateMonitorNotifier(ctx, deps.service.db, &models.CreateMonitorNotifierParams{
		MonitorID: deps.fixtures.Monitor.ID,
		Type:      models.NotifierEmail,
	})
	require.NoError(t, err)

	sender := &notificationSenderCapture{}
	deps.service.notificationSender = sender
	deps.service.llm = &stubLLMWorkflows{
		checkResult: &models.CheckResultWithSchema{
			CheckResultBase: models.CheckResultBase{
				Success:             true,
				DifferentToPrevious: true,
				Updates: models.MonitorUpdateDataList{
					{
						Headline: "{{Title}}",
						Subtitle: "Released at {{Link}}",
						Fields: models.MonitorUpdateFields{
							{
								MonitorSchemaField: models.MonitorSchemaField{
									Type: models.MonitorSchemaFieldTypeText,
									Name: "Title",
								},
								Value: "Example release",
							},
							{
								MonitorSchemaField: models.MonitorSchemaField{
									Type: models.MonitorSchemaFieldTypeURL,
									Name: "Link",
								},
								Value: "https://example.com/releases/1",
							},
						},
					},
				},
			},
		},
	}

	err = deps.service.PerformMonitorCheck(ctx, deps.fixtures.User.ID, deps.fixtures.Check, false)
	require.NoError(t, err)

	require.Len(t, sender.calls, 1)
	require.Equal(t, []models.Notifier{models.NotifierEmail}, sender.calls[0].NotificationChannels)
	require.Equal(t, deps.fixtures.Monitor.ID, sender.calls[0].Message.Monitor.ID)
	require.Equal(t, deps.fixtures.Monitor.ID, sender.calls[0].Message.New.MonitorID)
	require.Equal(t, "Example release", sender.calls[0].Message.New.Data.Fields.GetValue("Title"))
}

type stubLLMWorkflows struct {
	checkResult *models.CheckResultWithSchema
	triage      *llm.TriagerResponse
}

var _ llmWorkflowBuilder = &stubLLMWorkflows{}

func (s *stubLLMWorkflows) NewCheckWorkflow() llm.CheckWorkflowRunner {
	return stubCheckWorkflow{result: s.checkResult}
}

func (s *stubLLMWorkflows) NewTriageWorkflow() llm.TriageWorkflowRunner {
	return stubTriageWorkflow{response: s.triage}
}

type stubCheckWorkflow struct {
	result *models.CheckResultWithSchema
}

func (s stubCheckWorkflow) Run(ctx context.Context, params *llm.CheckParams) (*models.CheckResultWithSchema, error) {
	return s.result, nil
}

type stubTriageWorkflow struct {
	response *llm.TriagerResponse
}

func (s stubTriageWorkflow) Run(ctx context.Context, params *llm.CheckParams) (*llm.TriagerResponse, error) {
	if s.response != nil {
		return s.response, nil
	}
	return &llm.TriagerResponse{Approved: true}, nil
}
