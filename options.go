package client

import (
	"context"
	"io"
	"time"
)

type submissionKind string

const (
	m1     submissionKind = "m1"
	m2     submissionKind = "m2"
	m3     submissionKind = "m3"
	m4     submissionKind = "m4"
	final  submissionKind = "final"
	custom submissionKind = "custom"
)

// Options ...
type Options struct {
	ctx                 context.Context
	directory           string
	buildFilePath       string
	buildFileBaseName   string
	isSubmission        bool
	profilePath         string
	ratelimit           time.Duration
	stdout              io.WriteCloser
	stderr              io.WriteCloser
	jobQueueName        string
	submissionKind      submissionKind
	customSubmissionTag string
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

func SubmissionM1() Option {
	return func(o *Options) {
		o.submissionKind = m1
		o.isSubmission = true
	}
}

func SubmissionM2() Option {
	return func(o *Options) {
		o.submissionKind = m2
		o.isSubmission = true
	}
}

func SubmissionM3() Option {
	return func(o *Options) {
		o.submissionKind = m3
		o.isSubmission = true
	}
}

func SubmissionM4() Option {
	return func(o *Options) {
		o.submissionKind = m4
		o.isSubmission = true
	}
}

func SubmissionFinal() Option {
	return func(o *Options) {
		o.submissionKind = final
		o.isSubmission = true
	}
}

func SubmissionCustom(tag string) Option {
	return func(o *Options) {
		o.submissionKind = custom
		o.isSubmission = true
		o.customSubmissionTag = tag
	}
}

// JobQueueName ...
func JobQueueName(s string) Option {
	return func(o *Options) {
		o.jobQueueName = s
	}
}
