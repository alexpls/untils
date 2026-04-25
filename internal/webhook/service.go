package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/alexpls/untils/internal/db"
	"github.com/alexpls/untils/internal/models"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type Service struct {
	db         db.DB
	queries    *models.Queries
	validate   *validator.Validate
	httpClient *httpClient
}

func NewService(queries *models.Queries, db db.DB, validate *validator.Validate) *Service {
	return &Service{
		db:         db,
		queries:    queries,
		validate:   validate,
		httpClient: newHttpClient(),
	}
}

func (s *Service) ListWebhookTargets(ctx context.Context, userID int64) ([]*models.WebhookTarget, error) {
	return s.queries.ListWebhookTargets(ctx, s.db, userID)
}

type CreateWebhookTargetParams struct {
	UserID int64  `validate:"required"`
	URL    string `validate:"required,url"`
}

var ErrWebhookTargetAlreadyExists = errors.New("webhook target with this URL already exists")

func (s *Service) CreateWebhookTarget(ctx context.Context, params CreateWebhookTargetParams) error {
	if err := s.validate.Struct(params); err != nil {
		return err
	}

	// does the webhook already exist? not worth adding a db level uniqueness constraint here
	// (creating webhooks doesn't happen often, and if we get dupes it's not a big deal)
	// so doing the check application side.

	return db.WithTx(s.db, ctx, func(tx pgx.Tx) error {
		_, err := s.queries.GetWebhookTargetByURL(ctx, tx, &models.GetWebhookTargetByURLParams{
			UserID: params.UserID,
			Url:    pgtype.Text{String: params.URL, Valid: true},
		})
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("checking if webhook target already exists: %w", err)
		}
		if err == nil {
			return ErrWebhookTargetAlreadyExists
		}

		err = s.queries.CreateWebhookTarget(ctx, tx, &models.CreateWebhookTargetParams{
			UserID: params.UserID,
			Url:    pgtype.Text{String: params.URL, Valid: true},
		})
		if err != nil {
			return fmt.Errorf("creating webhook target: %w", err)
		}

		return nil
	})
}

func (s *Service) DeleteWebhookTarget(ctx context.Context, userID, id int64) error {
	return s.queries.DeleteWebhookTarget(ctx, s.db, &models.DeleteWebhookTargetParams{
		UserID:          userID,
		WebhookTargetID: id,
	})
}

func (s *Service) TestWebhookTarget(ctx context.Context, userID, id int64) (HttpResponse, error) {
	wh, err := s.queries.GetWebhookTarget(ctx, s.db, &models.GetWebhookTargetParams{
		UserID:          userID,
		WebhookTargetID: id,
	})
	if err != nil {
		return HttpResponse{}, fmt.Errorf("getting webhook target: %w", err)
	}

	message := NewMessageTest()
	jsonStr, err := json.Marshal(message)
	if err != nil {
		return HttpResponse{}, fmt.Errorf("marshaling json: %w", err)
	}

	return s.httpClient.Request(ctx, wh.Url.String, bytes.NewReader(jsonStr))
}

func (s *Service) SendMonitorNewResult(ctx context.Context, args SendArgs) (HttpResponse, error) {
	wh, err := s.queries.GetWebhookTarget(ctx, s.db, &models.GetWebhookTargetParams{
		UserID:          args.UserID,
		WebhookTargetID: args.WebhookTargetID,
	})
	if err != nil {
		return HttpResponse{}, fmt.Errorf("getting webhook target: %w", err)
	}

	monitor, err := s.queries.GetMonitor(ctx, s.db, &models.GetMonitorParams{
		UserID: args.UserID,
		ID:     args.MonitorID,
	})
	if err != nil {
		return HttpResponse{}, fmt.Errorf("getting monitor: %w", err)
	}

	newResultIDs := args.normalizedNewResultIDs()
	if len(newResultIDs) == 0 {
		return HttpResponse{}, fmt.Errorf("webhook send requires at least one new result id")
	}

	newResults := make([]models.MonitorResult, len(newResultIDs))
	for i, resultID := range newResultIDs {
		newResult, err := s.queries.GetMonitorResult(ctx, s.db, &models.GetMonitorResultParams{
			MonitorID: args.MonitorID,
			ResultID:  resultID,
		})
		if err != nil {
			return HttpResponse{}, fmt.Errorf("getting new result: %w", err)
		}
		newResults[i] = *newResult
	}

	oldResult := models.MonitorResult{Headline: "(none)"}
	if args.OldResultID != 0 {
		oldResultPtr, err := s.queries.GetMonitorResult(ctx, s.db, &models.GetMonitorResultParams{
			MonitorID: args.MonitorID,
			ResultID:  args.OldResultID,
		})
		if err != nil {
			return HttpResponse{}, fmt.Errorf("getting old result: %w", err)
		}
		oldResult = *oldResultPtr
	}

	jsonStr, err := MarshalMessageMonitorNewResults(*monitor, newResults, oldResult)
	if err != nil {
		return HttpResponse{}, fmt.Errorf("marshaling json: %w", err)
	}

	return s.httpClient.Request(ctx, wh.Url.String, bytes.NewReader(jsonStr))
}
