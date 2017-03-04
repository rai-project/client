package client

import (
	"github.com/k0kubun/pp"
	"github.com/rai-project/config"
	"github.com/rai-project/vipertags"
)

type clientConfig struct {
	UploadBucketName           string `json:"upload_bucket" config:"client.upload_bucket" default:"files.rai-project.com"`
	UploadDestinationDirectory string `json:"upload_destination_directory" config:"client.upload_destination_directory" default:"userdata"`
	BuildFileBaseName          string `json:"build_file" config:"client.build_file" default:"default"`
}

var (
	Config = &clientConfig{}
)

func (clientConfig) ConfigName() string {
	return "Client"
}

func (clientConfig) SetDefaults() {
}

func (a *clientConfig) Read() {
	vipertags.Fill(a)
	if a.BuildFileBaseName == "" || a.BuildFileBaseName == "default" {
		a.BuildFileBaseName = config.App.Name + "_build"
	}
}

func (c clientConfig) String() string {
	return pp.Sprintln(c)
}

func (c clientConfig) Debug() {
	log.Debug("Client Config = ", c)
}

func init() {
	config.Register(Config)
}
