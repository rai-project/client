// +build ece408ProjectMode

package client

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type submissionKind string
type submissionKindKey struct{}
type isSubmissionKey struct{}

const (
	m1        submissionKind = "m1"
	m2        submissionKind = "m2"
	m3        submissionKind = "m3"
	m4        submissionKind = "m4"
	final     submissionKind = "final"
	custom    submissionKind = "custom"
	algorithm submissionKind = "algorithm"
)

func SubmissionUploadExpiration() time.Time {
	return time.Now().AddDate(0, 6, 0) // six months from now
}

func validateSubmission(s submissionKind) error {
	for _, e := range validSubmissions {
		if strings.ToLower(string(s)) == strings.ToLower(string(e)) {
			return nil
		}
	}
	return &ValidationError{
		Message: fmt.Sprintf("invalid submission name. Valid submission names are %v", validSubmissions),
	}
}

func SubmissionName(s0 string) Option {
	return func(o *Options) {
		s := submissionKind(s0)
		if err := validateSubmission(s); err != nil {
			panic(err)
		}
		o.ctx = context.WithValue(o.ctx, submissionKindKey{}, s)
		o.ctx = context.WithValue(o.ctx, isSubmissionKey{}, true)
		o.ctx = context.WithValue(o.ctx, uploadExpirationKey{}, SubmissionUploadExpiration())
	}
}
