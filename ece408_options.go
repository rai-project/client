// +build ece408ProjectMode

package client

import "strings"

func isValidSubmission(s string) bool {
	for _, e := range validSubmissions {
		if strings.ToLower(s) == strings.ToLower(e) {
			return true
		}
	}
	panic(
		&ValidationError{
			Message: "invalid submission name. Valid submission names are " + strings.Join(validSubmissions, ", "),
		},
	)
	return false
}

func SubmissionName(s string) Option {
	return func(o *Options) {
		isValidSubmission(s)
		o.submissionKind = s
		o.isSubmission = true
	}
}
