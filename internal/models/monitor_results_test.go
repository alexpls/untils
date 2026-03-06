package models

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMonitorResultCanApplyCorrection(t *testing.T) {
	t.Parallel()

	result := MonitorResult{ID: 2}

	require.True(t, result.CanApplyCorrection(&MonitorResult{ID: 2}))
	require.False(t, result.CanApplyCorrection(&MonitorResult{ID: 3}))
	require.False(t, result.CanApplyCorrection(nil))
}

func TestLatestVisiblePreviousResultSkipsHiddenResults(t *testing.T) {
	t.Parallel()

	visible := &GetPreviousResultsWithCheckRow{
		MonitorResult: MonitorResult{ID: 2, Hidden: false},
	}
	hidden := &GetPreviousResultsWithCheckRow{
		MonitorResult: MonitorResult{ID: 1, Hidden: true},
	}

	require.Equal(t, visible, LatestVisiblePreviousResult([]*GetPreviousResultsWithCheckRow{
		hidden,
		visible,
	}))
	require.Nil(t, LatestVisiblePreviousResult([]*GetPreviousResultsWithCheckRow{hidden}))
}
