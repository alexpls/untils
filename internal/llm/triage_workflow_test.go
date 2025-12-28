//go:build integration

package llm

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTriageWorkflow(t *testing.T) {
	svc := newServiceForTest(t)
	triage := NewTriageWorkflow(svc)
	res, err := triage.Run(t.Context(), &TriageParams{
		Subject:      "Latest game that IGN has given a 10/10 rating",
		Instructions: "Hardware doesn't count.",
	})
	require.NoError(t, err)

	t.Logf("Check: %+v", res.Check)
	t.Logf("Triager: %+v", res.Triager)
}
