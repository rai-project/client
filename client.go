package client

import (
	"io"
	"os"
	"strings"
	"time"

	"path/filepath"

	"fmt"

	"github.com/Unknwon/com"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/fatih/color"
	colorable "github.com/mattn/go-colorable"
	"github.com/pkg/errors"
	"github.com/rai-project/archive"
	"github.com/rai-project/auth"
	"github.com/rai-project/auth/auth0"
	"github.com/rai-project/auth/secret"
	"github.com/rai-project/aws"
	"github.com/rai-project/broker"
	"github.com/rai-project/broker/sqs"
	"github.com/rai-project/config"
	"github.com/rai-project/pubsub"
	"github.com/rai-project/pubsub/redis"
	"github.com/rai-project/ratelimit"
	"github.com/rai-project/serializer/json"
	"github.com/rai-project/store"
	"github.com/rai-project/store/s3"
	"github.com/rai-project/uuid"
	"github.com/spf13/viper"
)

type client struct {
	ID          string
	uploadKey   string
	awsSession  *session.Session
	options     Options
	broker      broker.Broker
	pubsubConn  pubsub.Connection
	profile     auth.Profile
	isConnected bool
	subscribers []pubsub.Subscriber
}

type nopWriterCloser struct {
	io.Writer
}

func (nopWriterCloser) Close() error { return nil }

var (
	DefaultUploadExpiration = time.Now().AddDate(0, 0, 1) // tomorrow
)

func New(opts ...Option) (*client, error) {
	out, err := colorable.NewColorableStdout(), colorable.NewColorableStderr()
	if viper.GetBool("app.color") {
		out = colorable.NewNonColorable(out)
		err = colorable.NewNonColorable(err)
	}

	options := Options{
		directory:         "",
		isSubmission:      false,
		buildFileBaseName: Config.BuildFileBaseName,
		ratelimit:         ratelimit.Config.RateLimit,
		profilePath:       auth.DefaultProfilePath,
		stdout:            nopWriterCloser{out},
		stderr:            nopWriterCloser{err},
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

	if !config.IsDebug {
		if err := ratelimit.New(ratelimit.Limit(options.ratelimit)); err != nil {
			return err
		}
	}

	// Create an AWS session
	session, err := aws.NewSession(
		aws.Region(aws.AWSRegionUSEast1),
		aws.EncryptedAccessKey(aws.Config.AccessKey),
		aws.EncryptedSecretKey(aws.Config.SecretKey),
		aws.Sts(c.ID),
	)
	if err != nil {
		return err
	}
	c.awsSession = session

	return nil
}

func (c *client) resultHandler(msgs <-chan pubsub.Message) error {
	go func() {
		for msg := range msgs {
			var data string
			err := msg.Unmarshal(&data)
			if err != nil {
				continue
			}
			fmt.Print(data)
		}
	}()
	return nil
}

func (c *client) Upload() error {
	if c.awsSession == nil {
		log.Fatal("Expecting the awsSession to be set. Call Init before calling Upload")
		return errors.New("invalid usage")
	}

	st, err := s3.New(
		s3.Session(c.awsSession),
		store.Bucket(Config.UploadBucketName),
	)
	if err != nil {
		return err
	}

	fmt.Fprintln(c.options.stdout, color.YellowString("✱ Preparing your project directory for upload."))

	zippedReader, err := archive.Zip(c.options.directory)
	if err != nil {
		return err
	}
	defer zippedReader.Close()

	fmt.Fprintln(c.options.stdout, color.YellowString("✱ Uploading your project directory. This may take a few minutes."))

	uploadKey := Config.UploadDestinationDirectory + "/" + c.ID + ".tar." + archive.Config.CompressionFormatString

	key, err := st.UploadFrom(
		zippedReader,
		uploadKey,
		s3.Expiration(DefaultUploadExpiration),
		s3.Metadata(map[string]interface{}{
			"id":         c.ID,
			"profile":    c.profile.Info(),
			"created_at": time.Now(),
		}),
		s3.ContentType(archive.MimeType()),
		store.UploadProgressOutput(c.options.stdout),
		store.UploadProgressFinishMessage(color.GreenString("✱ Folder uploaded. Server is now processing your submission.")),
	)
	if err != nil {
		return err
	}
	c.uploadKey = key

	return nil
}

func (c *client) PublishSubscribe() error {
	brkr, err := sqs.New(
		sqs.QueueName(config.App.Name),
		broker.Serializer(json.New()),
		sqs.Session(c.awsSession),
	)
	if err != nil {
		return err
	}
	c.broker = brkr

	profile := c.profile.Info()

	err = brkr.Publish(
		config.App.Name,
		&broker.Message{
			ID: c.ID,
			Header: map[string]string{
				"id":              c.ID,
				"upload_key":      c.uploadKey,
				"username":        profile.Username,
				"user_access_key": profile.AccessKey,
				"user_secret_key": profile.SecretKey,
			},
			Body: []byte("data"),
		},
	)
	if err != nil {
		return err
	}

	fmt.Fprintln(c.options.stdout, color.GreenString("✱ Your job request has been posted to the queue."))

	redisConn, err := redis.New()
	if err != nil {
		return errors.Wrap(err, "cannot create a redis connection")
	}
	subscriber, err := redis.NewSubscriber(redisConn, config.App.Name+"/log-"+c.ID)
	if err != nil {
		return errors.Wrap(err, "cannot create redis subscriber")
	}

	msgs := subscriber.Start()
	c.resultHandler(msgs)

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
		sub.Stop()
	}

	if c.pubsubConn != nil {
		c.pubsubConn.Close()
	}

	return c.broker.Disconnect()
}

func (c *client) authenticate(profilePath string) error {

	fmt.Fprintln(c.options.stdout, color.GreenString("✱ Checking your athentication credentials."))

	var err error
	var prof auth.Profile

	provider := auth.Provider(strings.ToLower(auth.Config.Provider))
	switch provider {
	case auth.Auth0Provider:
		prof, err = auth0.NewProfile(auth.ProfilePath(profilePath))
	case auth.SecretProvider:
		prof, err = secret.NewProfile(auth.ProfilePath(profilePath))
	default:
		err = errors.Errorf("the auth provider %v specified is not supported", provider)
	}
	if err != nil {
		return err
	}

	ok, err := prof.Verify()
	if err != nil {
		return err
	}
	if !ok {
		return errors.Errorf("cannot authenticate using the credentials in %v", profilePath)
	}
	c.profile = prof
	return nil
}
