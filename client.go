package client

import (
	"os"

	"path/filepath"

	"github.com/Unknwon/com"
	"github.com/pkg/errors"
	"github.com/rai-project/ratelimit"
	"github.com/rai-project/user"
)

type client struct {
	options Options
}

func New(opts ...Option) (*client, error) {
	options := Options{
		directory:         "",
		isSubmission:      false,
		buildFileBaseName: Config.BuildFileBaseName,
		ratelimit:         ratelimit.Config.RateLimit,
		profilePath:       user.DefaultProfilePath,
		stdout:            os.Stdout,
		stderr:            os.Stderr,
	}

	for _, o := range opts {
		o(&options)
	}

	if options.directory == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, errors.Wrap(err, "cannot find current working directory")
		}
		options.directory = cwd
	}

	// Authenticate user
	if err := authenticate(options.profilePath); err != nil {
		return nil, err
	}

	buildFilePath := filepath.Join(options.directory, options.buildFileBaseName)
	if !com.IsFile(buildFilePath) {
		return nil, errors.Errorf("the build file [%v] does not exist", buildFilePath)
	}

	if err := ratelimit.New(ratelimit.Limit(options.ratelimit)); err != nil {
		return nil, err
	}

	return nil, nil
}
