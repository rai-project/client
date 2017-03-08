package client

import (
	"io"
	"os"
	"runtime"
	"time"

	"path/filepath"

	"fmt"

	"path"

	"github.com/Unknwon/com"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/cheggaaa/pb"
	"github.com/fatih/color"
	colorable "github.com/mattn/go-colorable"
	"github.com/pkg/errors"
	"github.com/rai-project/archive"
	"github.com/rai-project/aws"
	"github.com/rai-project/broker"
	"github.com/rai-project/broker/sqs"
	"github.com/rai-project/config"
	"github.com/rai-project/ratelimit"
	"github.com/rai-project/serializer/json"
	"github.com/rai-project/store"
	"github.com/rai-project/store/s3"
	"github.com/rai-project/user"
	"github.com/rai-project/uuid"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
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
		profilePath:       user.DefaultProfilePath,
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

func (c *client) resultHandler(pub broker.Publication) error {
	// msg := pub.Message()
	// pp.Println("body = ", string(msg.Body))
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

	fmt.Fprintln(c.options.stdout, color.YellowString("✱ Preparing your project directory for upload."))

	zippedReader, err := archive.Zip(c.options.directory)
	if err != nil {
		return err
	}
	defer zippedReader.Close()

	fmt.Fprintln(c.options.stdout, color.YellowString("✱ Uploading your project directory. This may take a few minutes."))

	// create progress reader
	memoryFileSystem := afero.NewMemMapFs()
	file, err := memoryFileSystem.Create(c.ID + ".tar." + archive.Config.CompressionFormatString)
	if err != nil {
		log.WithError(err).Errorf("cannot create %v memory fs", file.Name())
		return err
	}
	writtenBytes, err := io.Copy(file, zippedReader)
	if err != nil {
		log.WithError(err).Errorf("cannot write %v memory fs", file.Name())
		return err
	}
	defer file.Close()

	// move to the start of the file
	_, err = file.Seek(0, 0)
	if err != nil {
		log.WithError(err).Errorf("cannot seek to start of %v memory fs", file.Name())
		return err
	}

	// get the new original progress bar.
	bar := pb.New64(writtenBytes)

	// Set new human friendly print units.
	bar.SetUnits(pb.U_BYTES)

	// Set output to be stdout
	bar.Output = c.options.stdout

	// Show current speed is true.
	bar.ShowSpeed = true

	// Use different unicodes for Linux, OS X and Windows.
	switch runtime.GOOS {
	case "linux":
		// Need to add '\x00' as delimiter for unicode characters.
		bar.Format("┃\x00▓\x00█\x00░\x00┃")
	case "darwin":
		// Need to add '\x00' as delimiter for unicode characters.
		bar.Format(" \x00▓\x00 \x00░\x00 ")
	default:
		// Default to non unicode characters.
		bar.Format("[=> ]")
	}

	bar.Start()
	defer bar.FinishPrint(color.GreenString("✱ Folder uploaded. Server is now processing your submission."))

	reader := bar.NewProxyReader(file)

	uploadKey := Config.UploadDestinationDirectory + "/" + path.Base(file.Name())

	key, err := store.UploadFrom(
		reader,
		uploadKey,
		s3.Expiration(DefaultUploadExpiration),
		s3.Metadata(map[string]interface{}{
			"id":         c.ID,
			"profile":    c.profile,
			"created_at": time.Now(),
		}),
		s3.ContentType(archive.MimeType()),
	)
	if err != nil {
		return err
	}
	c.uploadKey = key

	return nil
}

func (c *client) Init() error {
	brkr, err := sqs.New(
		sqs.QueueName(config.App.Name),
		broker.Serializer(json.New()),
		sqs.Session(c.awsSession),
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

	fmt.Fprintln(c.options.stdout, color.GreenString("✱ Your job request has been posted to the queue."))

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

	fmt.Fprintln(c.options.stdout, color.GreenString("✱ Checking your athentication credentials."))

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
