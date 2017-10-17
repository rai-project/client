package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClientConfig(t *testing.T) {
	assert.NotEmpty(t, Config.SubmitRequirements)
	assert.IsType(t, []string{}, Config.SubmitRequirements)
}
