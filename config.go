package client

import (
	"runtime"

	"github.com/k0kubun/pp"
	"github.com/rai-project/config"
	"github.com/rai-project/vipertags"
)

type clientConfig struct {
	UploadBucketName           string        `json:"upload_bucket" config:"client.upload_bucket" default:"files.rai-project.com"`
	UploadDestinationDirectory string        `json:"upload_destination_directory" config:"client.upload_destination_directory" default:"userdata"`
	BuildFileBaseName          string        `json:"build_file" config:"client.build_file" default:"default"`
	SubmitRequirements         []string      `json:"submit_requirements" config:"client.submit_requirements"`
	JobQueueName               string        `json:"job_queue_name" config:"client.job_queue_name"`
	done                       chan struct{} `json:"-" config:"-"`
}

// Config ...
var (
	Config = &clientConfig{
		done: make(chan struct{}),
	}
)

// ConfigName ...
func (clientConfig) ConfigName() string {
	return "Client"
}

// SetDefaults ...
func (a *clientConfig) SetDefaults() {
	vipertags.SetDefaults(a)
}

// Read ...
func (a *clientConfig) Read() {
	defer close(a.done)
	vipertags.Fill(a)
	if a.BuildFileBaseName == "" || a.BuildFileBaseName == "default" {
		a.BuildFileBaseName = config.App.Name + "_build"
	}
	if a.JobQueueName == "" || a.JobQueueName == "default" {
		a.JobQueueName = config.App.Name + "_" + runtime.GOARCH
	}
}

// Wait ...
func (c clientConfig) Wait() {
	<-c.done
}

// String ...
func (c clientConfig) String() string {
	return pp.Sprintln(c)
}

// Debug ...
func (c clientConfig) Debug() {
	log.Debug("Client Config = ", c)
}

func init() {
	config.Register(Config)
}
