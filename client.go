package client

import (
	"context"
	"io"
	"os"
	"strings"
	"time"

	"github.com/Unknwon/com"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	colorable "github.com/mattn/go-colorable"
	"github.com/pkg/errors"
	"github.com/rai-project/archive"
	"github.com/rai-project/auth"
	"github.com/rai-project/auth/provider"
	"github.com/rai-project/broker"
	"github.com/rai-project/broker/rabbitmq"
	"github.com/rai-project/broker/sqs"
	"github.com/rai-project/config"
	"github.com/rai-project/database"
	"github.com/rai-project/model"
	"github.com/rai-project/pubsub"
	"github.com/rai-project/pubsub/redis"
	"github.com/rai-project/ratelimit"
	"github.com/rai-project/serializer"
	"github.com/rai-project/serializer/json"
	"github.com/rai-project/store"
	"github.com/rai-project/store/s3"
	"gopkg.in/mgo.v2/bson"
)

type Client struct {
	ID                    bson.ObjectId
	uploadKey             string
	awsSession            *session.Session
	mongodb               database.Database
	options               Options
	broker                broker.Broker
	pubsubConn            pubsub.Connection
	profile               auth.Profile
	serializer            serializer.Serializer
	subscribers           []pubsub.Subscriber
	buildSpec             model.BuildSpecification
	spinner               *spinner.Spinner
	configJobQueueName    string
	optionsJobQueueName   string
	buildFileJobQueueName string
	job                   *model.JobResponse
	jobBody               interface{}
	done                  chan bool
}

// DefaultUploadExpiration ...
var (
	DefaultUploadExpiration = func() time.Time {
		return time.Now().AddDate(0, 0, 7) // next week
	}
)

type uploadExpirationKey struct{}

// JobQueueName returns the job queue name from option, build file, or config in that order
func (c *Client) JobQueueName() string {

	if c.optionsJobQueueName != "" {
		return c.optionsJobQueueName
	}
	if c.buildFileJobQueueName != "" {
		return c.buildFileJobQueueName
	}
	return c.configJobQueueName
}

// New ...
func New(opts ...Option) (*Client, error) {

	out, err := colorable.NewColorableStdout(), colorable.NewColorableStderr()
	if !config.App.Color {
		out = colorable.NewNonColorable(out)
		err = colorable.NewNonColorable(err)
	}

	options := Options{
		ctx:               context.WithValue(context.Background(), uploadExpirationKey{}, DefaultUploadExpiration()),
		directory:         "",
		buildFileBaseName: Config.BuildFileBaseName,
		buildFilePath:     "",
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

	if err := ratelimit.New(ratelimit.Limit(options.ratelimit)); err != nil {
		return nil, err
	}

	clnt := &Client{
		ID:                  bson.NewObjectId(),
		options:             options,
		serializer:          json.New(),
		configJobQueueName:  Config.JobQueueName,
		optionsJobQueueName: options.jobQueueName,
		done:                make(chan bool),
	}

	return clnt, nil
}

func (c *Client) resultHandler(msgs <-chan pubsub.Message) error {

	parse := func(resp model.JobResponse) {
		c.parseLine(strings.TrimSpace(string(resp.Body)))
	}

	formatPrint := func(w io.WriteCloser, resp model.JobResponse) {
		if w == nil {
			return
		}
		body := strings.TrimSpace(string(resp.Body))
		if body == "" {
			return
		}
		if config.IsVerbose {
			fprint(w, "[ "+resp.CreatedAt.String()+"] ")
		}
		fprintln(w, body)
	}
	go func() {
		for msg := range msgs {
			var data model.JobResponse

			if c.spinner != nil {
				c.spinner.Stop()
				c.spinner = nil
			}
			err := msg.Unmarshal(&data)
			if err != nil {
				log.WithError(err).Debug("failed to unmarshal response data")
				continue
			}
			if data.Kind == model.StderrResponse {
				parse(data)
				formatPrint(c.options.stderr, data)
			} else if data.Kind == model.StdoutResponse {
				parse(data)
				formatPrint(c.options.stderr, data)
			}
		}
		c.done <- true
	}()
	return nil
}

// Upload ...
func (c *Client) Upload() error {
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

	fprintln(c.options.stdout, color.YellowString("✱ Preparing your project directory for upload."))

	dir := c.options.directory
	if !com.IsDir(dir) {
		return errors.Errorf("director %s not found", dir)
	}
	zippedReader, err := archive.Zip(dir)
	if err != nil {
		return err
	}
	defer zippedReader.Close()

	fprintln(c.options.stdout, color.YellowString("✱ Uploading your project directory. This may take a few minutes."))

	uploadKey := Config.UploadDestinationDirectory + "/" + c.ID.Hex() + "." + archive.Extension()

	uploadExpiration, ok := c.options.ctx.Value(uploadExpirationKey{}).(time.Time)
	if !ok {
		uploadExpiration = DefaultUploadExpiration()
	}

	key, err := st.UploadFrom(
		zippedReader,
		uploadKey,
		s3.Expiration(uploadExpiration),
		s3.Metadata(map[string]interface{}{
			"id":         c.ID,
			"type":       "user_upload",
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

// Publish ...
func (c *Client) Publish() error {

	profile := c.profile.Info()

	jobRequest := model.JobRequest{
		ID: c.ID,
		Base: model.Base{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		ClientVersion:      config.App.Version,
		UploadKey:          c.uploadKey,
		User:               profile.User,
		BuildSpecification: c.buildSpec,
	}

	body, err := c.serializer.Marshal(jobRequest)
	if err != nil {
		return err
	}

	// create a broker object.
	// this currently uses sqs, but should
	// be abstracted out
	var brkr broker.Broker
	if c.options.serverArch == "s390x" {
		brkr = rabbitmq.New(
			rabbitmq.QueueName(c.JobQueueName()),
			broker.Serializer(json.New()),
		)
	} else {
		brkr, err = sqs.New(
			sqs.QueueName(c.JobQueueName()),
			broker.Serializer(c.serializer),
			sqs.Session(c.awsSession),
		)
	}
	if err != nil {
		return err
	}
	c.broker = brkr

	brkr.Connect()
	log.Debug(color.GreenString("✱Submitting to queue= " + c.JobQueueName()))
	err = brkr.Publish(
		c.JobQueueName(),
		&broker.Message{
			ID: c.ID.Hex(),
			Header: map[string]string{
				"id":              c.ID.Hex(),
				"upload_key":      c.uploadKey,
				"username":        profile.Username,
				"user_access_key": profile.AccessKey,
				"user_secret_key": profile.SecretKey,
			},
			Body: body,
		},
	)
	if err != nil {
		return err
	}

	fprintln(c.options.stdout, color.GreenString("✱ Your job request has been posted to the queue."))

	if c.options.stdout != nil {
		c.spinner = spinner.New(spinner.CharSets[11], 100*time.Millisecond)
		c.spinner.Suffix = " Waiting for the server to process your request..."
		c.spinner.Writer = c.options.stdout
		c.spinner.Start()
	}

	return nil
}

// Subscribe ...
func (c *Client) Subscribe() error {
	redisConn, err := redis.New()
	if err != nil {
		return errors.Wrap(err, "cannot create a redis connection")
	}

	c.pubsubConn = redisConn

	// the channel name is of the form rai/log-xxxxxxxx
	subscribeChannel := config.App.Name + "/log-" + c.ID.Hex()
	subscriber, err := redis.NewSubscriber(redisConn, subscribeChannel)
	if err != nil {
		return errors.Wrap(err, "cannot create redis subscriber")
	}

	// run resultHandler for each message we get from
	// the pubsub
	c.resultHandler(subscriber.Start())

	c.subscribers = append(c.subscribers, subscriber)
	return nil
}

// Connect to the brokers
func (c *Client) Connect() error {
	if err := c.broker.Connect(); err != nil {
		return err
	}
	return nil
}

// Disconnect ...
func (c *Client) Disconnect() error {
	// stop subscribing to each of the subscribers
	// we have listened to
	for _, sub := range c.subscribers {
		sub.Stop()
	}
	// close the pubsub connection if it exists
	if c.pubsubConn != nil {
		c.pubsubConn.Close()
	}
	if c.broker != nil {
		return c.broker.Disconnect()
	}
	return nil
}

// Wait until we are complete (got the end signal)
func (c *Client) Wait() error {
	// the channel is written to (or closed)
	// when the end signal is received
	<-c.done
	return nil
}

func (c *Client) authenticate(profilePath string) error {

	fprintln(c.options.stdout, color.GreenString("✱ Checking your authentication credentials."))

	prof, err := provider.New(auth.ProfilePath(profilePath))
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
