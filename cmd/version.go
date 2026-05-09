package main

import "runtime/debug"

var buildRevision string
var readBuildInfo = debug.ReadBuildInfo

func buildVersion() string {
	if buildRevision != "" {
		return shortRevision(buildRevision)
	}

	buildInfo, ok := readBuildInfo()
	if !ok {
		return "dev"
	}

	for _, setting := range buildInfo.Settings {
		if setting.Key == "vcs.revision" {
			return "dev-" + shortRevision(setting.Value)
		}
	}

	return "dev"
}

func shortRevision(revision string) string {
	if len(revision) <= 7 {
		return revision
	}

	return revision[0:7]
}
