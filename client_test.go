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
		DisableRatelimit(),
	)
	assert.NoError(t, err)
	assert.NotNil(t, clt)

	err = clt.Validate()
	assert.NoError(t, err)

	err = clt.PublishSubscribe()
	assert.NoError(t, err)

	err = clt.Upload()
	assert.NoError(t, err)
	assert.NotEmpty(t, clt.uploadKey, "upload key must be set after upload")

	err = clt.Connect()
	assert.NoError(t, err)

	defer clt.Disconnect()
}

func TestMain(m *testing.M) {
	os.Setenv("DEBUG", "TRUE")
	os.Setenv("VERBOSE", "TRUE")
	config.Init()
	config.IsVerbose = true
	config.IsDebug = true
	os.Exit(m.Run())
}
