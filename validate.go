package client

import (
	"io/ioutil"
	"path/filepath"

	"github.com/Unknwon/com"
	"github.com/fatih/color"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/rai-project/config"
	"github.com/rai-project/database/mongodb"
)

type ValidationError struct {
	Message string
}

// Validate ...
func (c *Client) Validate() error {
	options := c.options

	// Authenticate user
	if err := c.authenticate(options.profilePath); err != nil {
		return err
	}

	var buf []byte

	if options.isSubmission {
		switch options.submissionKind {
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
		default:
			return errors.New("unrecognized submission type " + string(options.submissionKind))
		}
		fprintf(c.options.stdout, color.YellowString("Using the following build file for submission:\n%s"), string(buf))
	} else {
		var buildFilePath string
		if options.buildFilePath == "" {
			buildFilePath = filepath.Join(options.directory, options.buildFileBaseName+".yml")
		} else {
			buildFilePath = options.buildFilePath
		}
		if !com.IsFile(buildFilePath) {
			return errors.Errorf("the build file [%v] does not exist", buildFilePath)
		}
		if loc, err := filepath.Abs(buildFilePath); err == nil {
			buildFilePath = loc
		}
		var err error
		buf, err = ioutil.ReadFile(buildFilePath)
		if err != nil {
			return errors.Wrapf(err, "unable to read %v", buildFilePath)
		}
	}

	if err := yaml.Unmarshal(buf, &c.buildSpec); err != nil {
		return errors.Wrapf(err, "unable to parse build file")
	}

	if options.isSubmission {
		for _, requiredFileName := range Config.SubmitRequirements {
			requiredFilePath := filepath.Join(options.directory, requiredFileName)
			if !com.IsFile(requiredFilePath) {
				return errors.Errorf("Didn't find a file required for submission: [%v]", requiredFilePath)
			}
		}
	}

	// Fail early on no connection to submission database
	if options.isSubmission {
		db, err := mongodb.NewDatabase(config.App.Name)
		if err != nil {
			log.WithError(err).Error("Unable to contact submission database")
			return err
		}
		defer db.Close()
	}

	return nil
}
