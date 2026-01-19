//go:build integration

package llm

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTriageWorkflow(t *testing.T) {
	svc := newServiceForTest(t)

	triage := NewTriageWorkflow(svc)
	_, err := triage.Run(t.Context(), &CheckParams{
		Subject: "Latest game that IGN has given a 10/10 rating",
	})
	require.NoError(t, err)
}
