package client

import (
	"context"
	"io"
	"time"
)

// Options ...
type Options struct {
	ctx               context.Context
	directory         string
	buildFilePath     string
	buildFileBaseName string
	isSubmission      bool
	profilePath       string
	ratelimit         time.Duration
	stdout            io.WriteCloser
	stderr            io.WriteCloser
	jobQueueName      string
}

// Option ...
type Option func(*Options)

// Directory ...
func Directory(d string) Option {
	return func(o *Options) {
		o.directory = d
	}
}

// BuildFile ...
func BuildFilePath(s string) Option {
	return func(o *Options) {
		o.buildFilePath = s
	}
}

// BuildFileBaseName ...
func BuildFileBaseName(d string) Option {
	return func(o *Options) {
		o.buildFileBaseName = d
	}
}

// IsSubmission ...
func IsSubmission(d bool) Option {
	return func(o *Options) {
		o.isSubmission = d
	}
}

// ProfilePath ...
func ProfilePath(s string) Option {
	return func(o *Options) {
		o.profilePath = s
	}
}

// Ratelimit ...
func Ratelimit(d time.Duration) Option {
	return func(o *Options) {
		o.ratelimit = d
	}
}

// DisableRatelimit ...
func DisableRatelimit() Option {
	return Ratelimit(0)
}

// Stdout ...
func Stdout(s io.WriteCloser) Option {
	return func(o *Options) {
		o.stdout = s
	}
}

// Stderr ...
func Stderr(s io.WriteCloser) Option {
	return func(o *Options) {
		o.stderr = s
	}
}

// JobQueueName ...
func JobQueueName(s string) Option {
	return func(o *Options) {
		o.jobQueueName = s
	}
}
