package public

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAssetURL(t *testing.T) {
	assert.Equal(t, "/assets/not-fingerprinted/me.js", AssetURL("not-fingerprinted/me.js"))
	assert.Regexp(t, regexp.MustCompile(`^/assets/js/app-\w{32}\.js$`), AssetURL("js/app.js"))
}
