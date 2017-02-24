package client

import (
	"github.com/pkg/errors"
	"github.com/rai-project/user"
)

func authenticate(profilePath string) error {
	prof, err := user.NewProfile(profilePath)
	if err != nil {
		return err
	}
	ok := prof.Verify()
	if !ok {
		return errors.Errorf("cannot authenticate using the credentials in %v", profilePath)
	}
	return nil
}
