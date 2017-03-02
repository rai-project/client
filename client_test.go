package client

import (
	"os"
	"path/filepath"
	"testing"

	sourcepath "github.com/GeertJohan/go-sourcepath"
	"github.com/k0kubun/pp"
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
	defer clt.Disconnect()

	err = clt.Validate()
	assert.NoError(t, err)

	err = clt.Upload()
	assert.NoError(t, err)
	assert.NotEmpty(t, clt.uploadKey, "upload key must be set after upload")
	pp.Println(clt.uploadKey)
}

func TestMain(m *testing.M) {
	os.Setenv("DEBUG", "TRUE")
	os.Setenv("VERBOSE", "TRUE")
	config.Init()
	config.IsVerbose = true
	config.IsDebug = true
	os.Exit(m.Run())
}
