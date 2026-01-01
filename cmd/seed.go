package main

import (
	"context"
	"errors"
	"time"

	"github.com/alexpls/untils/internal/auth"
	"github.com/alexpls/untils/internal/db/sqlc"
	"github.com/alexpls/untils/internal/must"
)

func (a *app) seed() {
	must.True(a.config.env == "dev")

	ctx := context.Background()

	user, err := a.auth.CreateUser(ctx, "alexpls@fastmail.com", "abc123", "Australia/Brisbane")
	if err != nil {
		if errors.Is(err, auth.ErrUserExists) {
			return
		}
		panic(err)
	}

	a.seedMonitors(ctx, user.ID)
}

func (a *app) seedMonitors(ctx context.Context, userID int64) {
	now := time.Now()

	// Monitor 1: Latest Go version
	monitor1ID := a.seedMonitor(ctx, userID, "Latest Go version", "Check the official Go website for the latest stable release")

	// Add some completed checks and results for monitor 1
	check1Time := now.Add(-48 * time.Hour)
	check1ID := a.seedCheck(ctx, monitor1ID, sqlc.MonitorCheckStatusSuccess, check1Time, &check1Time)

	check2Time := now.Add(-24 * time.Hour)
	check2ID := a.seedCheck(ctx, monitor1ID, sqlc.MonitorCheckStatusSuccess, check2Time, &check2Time)

	// Add a scheduled check
	nextCheckTime := now.Add(24 * time.Hour)
	a.seedCheck(ctx, monitor1ID, sqlc.MonitorCheckStatusScheduled, nextCheckTime, nil)

	// Add results for monitor 1
	a.seedResult(ctx, monitor1ID, []int64{check1ID}, "Go 1.22.0", "2024-02-06", "Released", check1Time)
	a.seedResult(ctx, monitor1ID, []int64{check2ID}, "Go 1.23.0", "2024-08-13", "Released", check2Time)

	// Monitor 2: Taylor Swift next album
	monitor2ID := a.seedMonitor(ctx, userID, "Taylor Swift next album", "Look for official announcements about upcoming Taylor Swift albums")

	// Add checks for monitor 2
	check3Time := now.Add(-72 * time.Hour)
	check3ID := a.seedCheck(ctx, monitor2ID, sqlc.MonitorCheckStatusSuccess, check3Time, &check3Time)

	check4Time := now.Add(-24 * time.Hour)
	check4ID := a.seedCheck(ctx, monitor2ID, sqlc.MonitorCheckStatusSuccess, check4Time, &check4Time)

	// Failed check
	failedCheckTime := now.Add(-12 * time.Hour)
	a.seedFailedCheck(ctx, monitor2ID, failedCheckTime, "API rate limit exceeded")

	// Scheduled check
	nextCheck2Time := now.Add(12 * time.Hour)
	a.seedCheck(ctx, monitor2ID, sqlc.MonitorCheckStatusScheduled, nextCheck2Time, nil)

	// Results for monitor 2
	a.seedResult(ctx, monitor2ID, []int64{check3ID, check4ID}, "The Tortured Poets Department", "2024-04-19", "Released", check3Time)
}

func (a *app) seedMonitor(ctx context.Context, userID int64, subject, instructions string) int64 {
	var id int64
	err := a.db.QueryRow(ctx, `
		INSERT INTO monitors (user_id, subject, instructions, status, updated_at, created_at)
		VALUES ($1, $2, $3, 'active', now(), now())
		RETURNING id
	`, userID, subject, instructions).Scan(&id)
	if err != nil {
		panic(err)
	}
	return id
}

func (a *app) seedCheck(ctx context.Context, monitorID int64, status sqlc.MonitorCheckStatus, scheduledFor time.Time, doneAt *time.Time) int64 {
	var id int64
	err := a.db.QueryRow(ctx, `
		INSERT INTO monitor_checks (monitor_id, status, scheduled_for, done_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, monitorID, status, scheduledFor, doneAt).Scan(&id)
	if err != nil {
		panic(err)
	}
	return id
}

func (a *app) seedFailedCheck(ctx context.Context, monitorID int64, scheduledFor time.Time, failureReason string) int64 {
	var id int64
	doneAt := scheduledFor.Add(5 * time.Second)
	err := a.db.QueryRow(ctx, `
		INSERT INTO monitor_checks (monitor_id, status, scheduled_for, failure_reason, done_at)
		VALUES ($1, 'failed', $2, $3, $4)
		RETURNING id
	`, monitorID, scheduledFor, failureReason, doneAt).Scan(&id)
	if err != nil {
		panic(err)
	}
	return id
}

func (a *app) seedResult(ctx context.Context, monitorID int64, checkIDs []int64, result, date, pastTenseVerb string, createdAt time.Time) int64 {
	var id int64
	err := a.db.QueryRow(ctx, `
		INSERT INTO monitor_results (monitor_id, confirming_check_ids, result, date, date_past_tense_verb, citations, latest_confirmation_at, created_at)
		VALUES ($1, $2, $3, $4::date, $5, '[]'::jsonb, $6, $6)
		RETURNING id
	`, monitorID, checkIDs, result, date, pastTenseVerb, createdAt).Scan(&id)
	if err != nil {
		panic(err)
	}
	return id
}
