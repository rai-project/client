package client

import "time"
import "io"

type Options struct {
	directory         string
	buildFileBaseName string
	isSubmission      bool
	profilePath       string
	ratelimit         time.Duration
	stdout            io.WriteCloser
	stderr            io.WriteCloser
}

type Option func(*Options)

func Directory(d string) Option {
	return func(o *Options) {
		o.directory = d
	}
}

func BuildFileBaseName(d string) Option {
	return func(o *Options) {
		o.buildFileBaseName = d
	}
}

func IsSubmission(d bool) Option {
	return func(o *Options) {
		o.isSubmission = d
	}
}

func ProfilePath(s string) Option {
	return func(o *Options) {
		o.profilePath = s
	}
}

func Ratelimit(d time.Duration) Option {
	return func(o *Options) {
		o.ratelimit = d
	}
}

func DisableRatelimit() Option {
	return Ratelimit(0)
}

func Stdout(s io.WriteCloser) Option {
	return func(o *Options) {
		o.stdout = s
	}
}

func Stderr(s io.WriteCloser) Option {
	return func(o *Options) {
		o.stderr = s
	}
}
