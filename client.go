package client

import (
	"os"

	"path/filepath"

	"github.com/Unknwon/com"
	"github.com/k0kubun/pp"
	"github.com/pkg/errors"
	"github.com/rai-project/broker"
	"github.com/rai-project/broker/sqs"
	"github.com/rai-project/ratelimit"
	"github.com/rai-project/serializer/json"
	"github.com/rai-project/user"
	"github.com/rai-project/uuid"
)

type client struct {
	ID          string
	options     Options
	broker      broker.Broker
	subscribers []broker.Subscriber
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

	return &client{
		ID:      uuid.NewV4(),
		options: options,
	}, nil
}

func (c *client) Validate() error {
	options := c.options

	// Authenticate user
	if err := authenticate(options.profilePath); err != nil {
		return err
	}

	buildFilePath := filepath.Join(options.directory, options.buildFileBaseName)
	if !com.IsFile(buildFilePath) {
		return errors.Errorf("the build file [%v] does not exist", buildFilePath)
	}

	if err := ratelimit.New(ratelimit.Limit(options.ratelimit)); err != nil {
		return err
	}
	return nil
}

func (c *client) resultHandler(pub broker.Publication) error {
	msg := pub.Message()
	pp.Println(string(msg.Body))
	return nil
}

func (c *client) Init() error {
	brkr := sqs.New(
		broker.Serializer(json.New()),
	)
	c.broker = brkr

	err := brkr.Publish(
		"rai",
		&broker.Message{
			ID: c.ID,
			Header: map[string]string{
				"id": c.ID,
			},
			Body: []byte("data"),
		},
	)
	if err != nil {
		return err
	}

	subscriber, err := brkr.Subscribe(
		"log-"+c.ID,
		c.resultHandler,
		broker.AutoAck(true),
	)
	if err != nil {
		return err
	}

	c.subscribers = append(c.subscribers, subscriber)

	return nil
}

func (c *client) Connect() error {
	return c.broker.Connect()
}

func (c *client) Disconnect() error {
	for _, sub := range c.subscribers {
		sub.Unsubscribe()
	}
	return c.broker.Disconnect()
}

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
