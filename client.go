package client

import (
	"os"
	"time"

	"path/filepath"

	"github.com/Unknwon/com"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/k0kubun/pp"
	"github.com/pkg/errors"
	"github.com/rai-project/archive"
	"github.com/rai-project/aws"
	"github.com/rai-project/broker"
	"github.com/rai-project/broker/sqs"
	"github.com/rai-project/ratelimit"
	"github.com/rai-project/serializer/json"
	"github.com/rai-project/store"
	"github.com/rai-project/store/s3"
	"github.com/rai-project/user"
	"github.com/rai-project/uuid"
)

type client struct {
	ID          string
	uploadKey   string
	awsSession  *session.Session
	options     Options
	broker      broker.Broker
	profile     *user.Profile
	isConnected bool
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
		ID:          uuid.NewV4(),
		isConnected: false,
		options:     options,
	}, nil
}

func (c *client) Validate() error {
	options := c.options

	// Authenticate user
	if err := c.authenticate(options.profilePath); err != nil {
		return err
	}

	buildFilePath := filepath.Join(options.directory, options.buildFileBaseName+".yml")
	if !com.IsFile(buildFilePath) {
		return errors.Errorf("the build file [%v] does not exist", buildFilePath)
	}

	if err := ratelimit.New(ratelimit.Limit(options.ratelimit)); err != nil {
		return err
	}

	// Create an AWS session
	session, err := aws.NewSession(
		aws.Region(aws.AWSRegionUSEast1),
		aws.EncryptedAccessKey(aws.Config.AccessKey),
		aws.EncryptedSecretKey(aws.Config.SecretKey),
		aws.Sts(),
	)
	if err != nil {
		return err
	}
	c.awsSession = session

	return nil
}

func (c *client) resultHandler(pub broker.Publication) error {
	msg := pub.Message()
	pp.Println(string(msg.Body))
	return nil
}

func (c *client) Upload() error {
	if c.awsSession == nil {
		log.Fatal("Expecting the awsSession to be set. Call Init before calling Upload")
		return errors.New("invalid usage")
	}

	store, err := s3.New(
		s3.Session(c.awsSession),
		store.Bucket(Config.UploadBucketName),
	)
	if err != nil {
		return err
	}

	zippedReader, err := archive.Zip(c.options.directory)
	if err != nil {
		return err
	}
	defer zippedReader.Close()
	key, err := store.UploadFrom(
		zippedReader,
		Config.UploadDestinationDirectory+"/"+c.ID+".tar."+archive.Config.CompressionFormatString,
		s3.Expiration(time.Now().AddDate(0, 0, 1)), // tomorrow
	)
	if err != nil {
		return err
	}
	c.uploadKey = key

	return nil
}

func (c *client) Init() error {
	brkr, err := sqs.New(
		sqs.Session(c.awsSession),
		broker.Serializer(json.New()),
	)
	if err != nil {
		return err
	}
	c.broker = brkr

	err = brkr.Publish(
		"rai",
		&broker.Message{
			ID: c.ID,
			Header: map[string]string{
				"id":              c.ID,
				"upload_key":      c.uploadKey,
				"username":        c.profile.Username,
				"user_access_key": c.profile.AccessKey,
				"user_secret_key": c.profile.SecretKey,
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
	if err := c.broker.Connect(); err != nil {
		return err
	}
	c.isConnected = true
	return nil
}

func (c *client) Disconnect() error {
	if !c.isConnected {
		return nil
	}
	for _, sub := range c.subscribers {
		sub.Unsubscribe()
	}
	return c.broker.Disconnect()
}

func (c *client) authenticate(profilePath string) error {
	prof, err := user.NewProfile(profilePath)
	if err != nil {
		return err
	}
	ok := prof.Verify()
	if !ok {
		return errors.Errorf("cannot authenticate using the credentials in %v", profilePath)
	}
	c.profile = prof
	return nil
}
