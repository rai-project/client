package client

import (
	"os"
	"path/filepath"
	"testing"

	sourcepath "github.com/GeertJohan/go-sourcepath"
	"github.com/rai-project/config"
	"github.com/stretchr/testify/assert"
)

func TestClient(t *testing.T) {
	clt, err := New(
		Directory(filepath.Join(sourcepath.MustAbsoluteDir(), "_fixtures")),
		BuildFileBaseName("rai_build"),
	)
	assert.NoError(t, err)
	assert.NotNil(t, clt)
	defer clt.Disconnect()

	err = clt.Validate()
	assert.NoError(t, err)
}

func TestMain(m *testing.M) {
	os.Setenv("DEBUG", "TRUE")
	os.Setenv("VERBOSE", "TRUE")
	config.Init()
	config.IsVerbose = true
	config.IsDebug = true
	os.Exit(m.Run())
}
