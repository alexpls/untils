package notifications

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRenderMonitorNewResult(t *testing.T) {
	t.Parallel()

	rendered, err := RenderMonitorNewResult(context.Background(), MonitorNewResult{
		Subject: "Example monitor",
		New:     "new value",
		Old:     "old value",
	})
	require.NoError(t, err)

	require.Equal(t, "Monitor changed: Example monitor", rendered.Email.Subject)
	require.Equal(t, "Monitor changed: Example monitor", rendered.Pushover.Title)
	require.Equal(t, "New: new value\n\nOld: old value", rendered.Email.TextBody)
	require.Equal(t, "New: new value\n\nOld: old value", rendered.Pushover.Message)
	require.Contains(t, rendered.Email.HTMLBody, "Monitor changed: Example monitor")
	require.Contains(t, rendered.Email.HTMLBody, "new value")
	require.Contains(t, rendered.Email.HTMLBody, "old value")
}

func TestEmailTemplateStore(t *testing.T) {
	t.Parallel()

	store := NewEmailTemplateStore()

	templates := store.Templates()
	require.Len(t, templates, 1)
	require.Equal(t, "new_result", templates[0].Key)

	tmpl, ok := store.Template("new_result")
	require.True(t, ok)
	require.Equal(t, "New result", tmpl.Name)

	_, ok = store.Template("missing")
	require.False(t, ok)
}
