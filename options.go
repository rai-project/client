package client

import "time"
import "io"

type submissionKind string

const (
	final submissionKind = "final"
	m2    submissionKind = "m2"
	m3    submissionKind = "m3"
)

// Options ...
type Options struct {
	directory         string
	buildFileBaseName string
	isSubmission      bool
	profilePath       string
	ratelimit         time.Duration
	stdout            io.WriteCloser
	stderr            io.WriteCloser
	jobQueueName      string
	submissionKind    submissionKind
}

// Option ...
type Option func(*Options)

// Directory ...
func Directory(d string) Option {
	return func(o *Options) {
		o.directory = d
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

func SubmissionKindM2() Option {
	return func(o *Options) {
		o.submissionKind = m2
	}
}

func SubmissionKindM3() Option {
	return func(o *Options) {
		o.submissionKind = m3
	}
}

func SubmissionKindFinal() Option {
	return func(o *Options) {
		o.submissionKind = final
	}
}

// JobQueueName ...
func JobQueueName(s string) Option {
	return func(o *Options) {
		o.jobQueueName = s
	}
}
