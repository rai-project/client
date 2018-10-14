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
	m1     submissionKind = "m1"
	m2     submissionKind = "m2"
	m3     submissionKind = "m3"
	m4     submissionKind = "m4"
	final  submissionKind = "final"
	custom submissionKind = "custom"
)

func SubmissionUploadExpiration() time.Time {
	return time.Now().AddDate(0, 6, 0) // six months from now
}

func isValidSubmission(s submissionKind) bool {
	for _, e := range validSubmissions {
		if strings.ToLower(string(s)) == strings.ToLower(string(e)) {
			return true
		}
	}
	panic(
		&ValidationError{
			Message: fmt.Sprintf("invalid submission name. Valid submission names are %v", validSubmissions),
		},
	)
	return false
}

func SubmissionName(s0 string) Option {
	return func(o *Options) {
		s := submissionKind(s0)
		isValidSubmission(s)
		o.ctx = context.WithValue(o.ctx, submissionKindKey{}, s)
		o.ctx = context.WithValue(o.ctx, isSubmissionKey{}, true)
		o.ctx = context.WithValue(o.ctx, uploadExpirationKey{}, SubmissionUploadExpiration())
	}
}
