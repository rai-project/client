package client

import (
	"github.com/pkg/errors"
	"io/ioutil"
)

// validation of client parameters this includes:
//  - authentication, roles
//  - run custom prevalidation steps
//  - existance and validity of the build spec file
func (c *Client) Validate() error {

	options := c.options

	// Authenticate user using their profile
	if err := c.authenticate(options.profilePath); err != nil {
		return err
	}

	// For some modes, there might be some prevalidation steps
	// that must be performed after authentication. This includes
	// things like validity of their profile.
	if err := c.preValidate(); err != nil {
		return err
	}

	// Find the build sepc file. returns an error
	// if the file cannot be found
	if c.buildSpec.Commands.Build == nil {
		specFilePath, err := c.findSpecFile()
		if err != nil {
			return err
		}

		// Read the build spec file into a buffer
		buf, err := ioutil.ReadFile(specFilePath)
		if err != nil {
			return errors.Wrapf(err, "unable to read %v", specFilePath)
		}

		// Read the build spec file into our internal data structure
		if err := c.readSpec(buf); err != nil {
			return err
		}
	}
	return nil
}
