// +build ece408ProjectMode

package client

import (
	"fmt"
	"strings"
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
		o.submissionKind = s
		o.isSubmission = true
	}
}
