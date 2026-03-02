package instructions

import _ "embed"

//go:embed episodic_content.md
var episodicContentBody string

type episodicContent struct{}

func (e episodicContent) Name() string {
	return "Episodic content"
}

func (e episodicContent) Description() string {
	return "use for results that are episodic in nature (e.g. podcast episodes, tv show episodes)"
}

func (e episodicContent) Body() string {
	return episodicContentBody
}
