package session

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSessionData_SetAndPopFlash(t *testing.T) {
	data := SessionData{}

	data.SetFlash(FlashTypeAlert, "Password changed.")
	assert.Equal(t, "Password changed.", data.PopFlash(FlashTypeAlert))
	assert.Equal(t, "", data.PopFlash(FlashTypeAlert))
}
