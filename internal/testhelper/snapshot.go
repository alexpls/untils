package testhelper

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

var snapshotFlag = flag.Bool("snapshot", false, "whether to update snapshots")

func Snapshot(t *testing.T, name string, builder func() string) string {
	t.Helper()
	require.True(t, len(name) > 1, "name can't be blank")

	p := filepath.Join("./testdata", "snapshot_"+name)

	if *snapshotFlag {
		require.NoError(t, os.MkdirAll("./testdata", 0755))
		s := builder()
		require.NoError(t, os.WriteFile(p, []byte(s), 0644))
		return s
	} else {
		val, err := os.ReadFile(p)
		require.NoError(t, err, "snapshot file not found. run the tests with -args -snapshot to create snapshots")
		return string(val)
	}
}

func SnapshotMatch(t *testing.T, name string, actual string) {
	t.Helper()
	val := Snapshot(t, name, func() string {
		return actual
	})
	require.Equal(t, val, actual)
}
