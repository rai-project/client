// +build ece408ProjectMode

package client

import (
	"path/filepath"

	"github.com/Unknwon/com"
	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/rai-project/config"
	"github.com/rai-project/database/mongodb"
)

func (c *Client) validateUserRole() error {
	role, err := c.profile.GetRole()
	if err != nil {
		return err
	}

	if !isECE408Role(role) {
		return &ValidationError{
			Message: "You are using an invalid client. Please download the correct client from http://github.com/rai-project/rai",
		}
	}

	return nil
}

func (c *Client) validateSubmission() error {
	var buf []byte

	options := c.options

	submissionKind, ok := options.ctx.Value(submissionKindKey{}).(submissionKind)
	if !ok {
		return errors.New("invalid submission")
	}
	switch submissionKind {
	case m1:
		buf = m1Build
	case m2:
		buf = m2Build
	case m3:
		buf = m3Build
	case m4:
		buf = m4Build
	case final:
		buf = finalBuild
	case custom:
		log.Info("Using embedded eval build for custom submission")
		buf = evalBuild
	case algorithm:

	default:
		return errors.Errorf("unrecognized submission type %v", submissionKind)
	}
	fprintf(c.options.stdout, color.CyanString("✱ Using the following build file for submission:\n%s"), string(buf))

	if err := c.readSpec(buf); err != nil {
		return err
	}

	if submissionKind != algorithm {
		for _, requiredFileName := range Config.SubmitRequirements {
			requiredFilePath := filepath.Join(options.directory, requiredFileName)
			if !com.IsFile(requiredFilePath) {
				return errors.Errorf("Didn't find a file required for submission: [%v]", requiredFilePath)
			}
		}
	}

	// Fail early on no connection to submission database
	db, err := mongodb.NewDatabase(config.App.Name)
	if err != nil {
		log.WithError(err).Error("Unable to contact submission database")
		return err
	}
	defer db.Close()

	return nil
}

// Validate ...
func (c *Client) preValidate() error {
	if err := c.validateUserRole(); err != nil {
		return err
	}
	options := c.options
	isSubmission, ok := options.ctx.Value(isSubmissionKey{}).(bool)
	if !ok {
		return nil
	}
	if !isSubmission {
		return nil
	}

	return c.validateSubmission()
}
