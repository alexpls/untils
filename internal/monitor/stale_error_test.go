package monitor

import (
	"fmt"
	"testing"

	"github.com/alexpls/untils/internal/errortypes"
	"github.com/stretchr/testify/require"
)

func TestIsStaleMonitorWorkError(t *testing.T) {
	t.Parallel()

	require.True(t, isStaleMonitorWorkError(&errortypes.ResourceNotFoundError{Resource: "monitor", ID: 1}))
	require.True(t, isStaleMonitorWorkError(&errortypes.ErrSubjectMismatch{}))
	require.False(t, isStaleMonitorWorkError(fmt.Errorf("other error")))
}
