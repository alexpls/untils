package main

import (
	"runtime/debug"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildVersionUsesInjectedRevision(t *testing.T) {
	withBuildVersionInputs(t, "1234567890abcdef", []debug.BuildSetting{{Key: "vcs.revision", Value: "abcdef1234567890"}}, true)

	assert.Equal(t, "1234567", buildVersion())
}

func TestBuildVersionUsesDevVCSRevision(t *testing.T) {
	withBuildVersionInputs(t, "", []debug.BuildSetting{{Key: "vcs.revision", Value: "abcdef1234567890"}}, true)

	assert.Equal(t, "dev-abcdef1", buildVersion())
}

func TestBuildVersionFallsBackToDev(t *testing.T) {
	withBuildVersionInputs(t, "", nil, false)
	assert.Equal(t, "dev", buildVersion())

	withBuildVersionInputs(t, "", []debug.BuildSetting{}, true)
	assert.Equal(t, "dev", buildVersion())
}

func withBuildVersionInputs(t *testing.T, revision string, settings []debug.BuildSetting, ok bool) {
	t.Helper()

	previousBuildRevision := buildRevision
	previousReadBuildInfo := readBuildInfo
	buildRevision = revision
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{Settings: settings}, ok
	}

	t.Cleanup(func() {
		buildRevision = previousBuildRevision
		readBuildInfo = previousReadBuildInfo
	})
}
