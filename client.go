package client

//go:generate esc -o fixtures.go -pkg client _fixtures/rai_submission.yml

import (
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"path/filepath"

	"fmt"

	"github.com/Unknwon/com"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	colorable "github.com/mattn/go-colorable"
	"github.com/pkg/errors"
	"github.com/rai-project/archive"
	"github.com/rai-project/auth"
	"github.com/rai-project/auth/provider"
	"github.com/rai-project/aws"
	"github.com/rai-project/broker"
	"github.com/rai-project/broker/sqs"
	"github.com/rai-project/config"
	"github.com/rai-project/database"
	"github.com/rai-project/database/mongodb"
	"github.com/rai-project/model"
	"github.com/rai-project/pubsub"
	"github.com/rai-project/pubsub/redis"
	"github.com/rai-project/ratelimit"
	"github.com/rai-project/serializer"
	"github.com/rai-project/serializer/json"
	"github.com/rai-project/store"
	"github.com/rai-project/store/s3"
	"github.com/rai-project/uuid"
	"gopkg.in/mgo.v2/bson"
	"gopkg.in/yaml.v2"
)

type client struct {
	ID                    string
	uploadKey             string
	awsSession            *session.Session
	mongodb               database.Database
	options               Options
	broker                broker.Broker
	pubsubConn            pubsub.Connection
	profile               auth.Profile
	isConnected           bool
	serializer            serializer.Serializer
	subscribers           []pubsub.Subscriber
	buildSpec             model.BuildSpecification
	spinner               *spinner.Spinner
	configJobQueueName    string
	optionsJobQueueName   string
	buildFileJobQueueName string
	ranking               *model.Fa2017Ece408Ranking
	done                  chan bool
}

var (
	submissionName  = "_fixtures/rai_submission.yml"
	submissionBuild = FSMustByte(false, "/"+submissionName)
)

type nopWriterCloser struct {
	io.Writer
}

// Close ...
func (nopWriterCloser) Close() error { return nil }

// DefaultUploadExpiration ...
var (
	DefaultUploadExpiration = func() time.Time {
		return time.Now().AddDate(0, 0, 1) // tomorrow
	}
)

// JobQueueName returns the job queue name from option, build file, or config in that order
func (c *client) JobQueueName() string {
	if c.optionsJobQueueName != "" {
		return c.optionsJobQueueName
	} else if c.buildFileJobQueueName != "" {
		return c.buildFileJobQueueName
	}
	return c.configJobQueueName
}

// New ...
func New(opts ...Option) (*client, error) {
	out, err := colorable.NewColorableStdout(), colorable.NewColorableStderr()
	if !config.App.Color {
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
		ID:                  uuid.NewV4(),
		isConnected:         false,
		options:             options,
		serializer:          json.New(),
		configJobQueueName:  Config.JobQueueName,
		optionsJobQueueName: options.jobQueueName,
		done:                make(chan bool),
	}, nil
}

func (c *client) fixDockerPushCredentials() (err error) {
	profileInfo := c.profile.Info()
	if profileInfo.DockerHub == nil {
		return
	}

	buildImage := c.buildSpec.Commands.BuildImage
	if buildImage == nil {
		return
	}
	push := buildImage.Push
	if push == nil {
		return
	}
	if !push.Push {
		return
	}
	if push.Credentials.Username == "" && push.Credentials.Password == "" {
		push.Credentials.Username = profileInfo.DockerHub.Username
		push.Credentials.Password = profileInfo.DockerHub.Password
	}
	return
}

func (c *client) RecordIfSubmission() error {

	if !c.options.isSubmission {
		return nil
	}

	db, err := mongodb.NewDatabase(config.App.Name)
	if err != nil {
		return err
	}
	log.Info("Connected to submission database")
	defer db.Close()

	tbl, err := mongodb.NewTable(db, "rankings")
	if err != nil {
		return err
	}

	if c.ranking == nil {
		log.Error("submission ranking was not filled")
	}

	prof, err := provider.New()
	user := prof.Info()
	c.ranking.Username = user.Username
	c.ranking.Teamname = user.Team.Name
	c.ranking.CreatedAt = time.Now()

	err = tbl.Insert(c.ranking)
	log.Info("Inserted ranking ranking")

	return err
}

// Validate ...
func (c *client) Validate() error {
	options := c.options

	// Authenticate user
	if err := c.authenticate(options.profilePath); err != nil {
		return err
	}

	var buf []byte
	if options.isSubmission {
		buf = submissionBuild
		fmt.Fprintf(c.options.stdout, color.YellowString("Using the following build file for submission:\n%s"), string(buf))
	} else {
		buildFilePath := filepath.Join(options.directory, options.buildFileBaseName+".yml")
		if !com.IsFile(buildFilePath) {
			return errors.Errorf("the build file [%v] does not exist", buildFilePath)
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
				return errors.Errorf("Didn't find a required file: [%v]", requiredFilePath)
			}
		}
	}
	// set the queue from the build file
	c.buildFileJobQueueName = config.App.Name + "_" + c.buildSpec.Resources.CPU.Architecture
	log.Debug("inferring queue ", c.buildFileJobQueueName, " from build file. May be overrriden by client.Options")

	if !config.IsDebug {
		if err := ratelimit.New(ratelimit.Limit(options.ratelimit)); err != nil {
			return err
		}
	}

	if err := c.fixDockerPushCredentials(); err != nil {
		return err
	}

	// Create an AWS session
	session, err := aws.NewSession(
		aws.Region(aws.AWSRegionUSEast1),
		aws.AccessKey(aws.Config.AccessKey),
		aws.SecretKey(aws.Config.SecretKey),
		aws.Sts(c.ID),
	)
	if err != nil {
		return err
	}
	c.awsSession = session

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

func (c *client) resultHandler(msgs <-chan pubsub.Message) error {

	parse := func(w io.WriteCloser, resp model.JobResponse) {
		if c.ranking == nil {
			c.ranking = &model.Fa2017Ece408Ranking{}
			c.ranking.ID = bson.NewObjectId()
		}
		parseLine(c.ranking, strings.TrimSpace(string(resp.Body)))
	}

	formatPrint := func(w io.WriteCloser, resp model.JobResponse) {
		body := strings.TrimSpace(string(resp.Body))
		if body == "" {
			return
		}
		if config.IsVerbose {
			fmt.Fprint(w, "[ "+resp.CreatedAt.String()+"] ")
		}
		fmt.Fprintln(w, body)
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
				parse(c.options.stderr, data)
				formatPrint(c.options.stderr, data)
			} else if data.Kind == model.StdoutResponse {
				parse(c.options.stderr, data)
				formatPrint(c.options.stderr, data)
			}
		}
		c.done <- true
	}()
	return nil
}

// Upload ...
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

	dir := c.options.directory
	if !com.IsDir(dir) {
		return errors.Errorf("director %s not found", dir)
	}
	zippedReader, err := archive.Zip(dir)
	if err != nil {
		return err
	}
	defer zippedReader.Close()

	fmt.Fprintln(c.options.stdout, color.YellowString("✱ Uploading your project directory. This may take a few minutes."))

	uploadKey := Config.UploadDestinationDirectory + "/" + c.ID + "." + archive.Extension()

	key, err := st.UploadFrom(
		zippedReader,
		uploadKey,
		s3.Expiration(DefaultUploadExpiration()),
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
func (c *client) Publish() error {

	profile := c.profile.Info()

	jobRequest := model.JobRequest{
		Base: model.Base{
			ID:        c.ID,
			CreatedAt: time.Now(),
		},
		UploadKey:          c.uploadKey,
		User:               profile.User,
		BuildSpecification: c.buildSpec,
	}

	body, err := c.serializer.Marshal(jobRequest)
	if err != nil {
		return err
	}

	brkr, err := sqs.New(
		sqs.QueueName(c.JobQueueName()),
		broker.Serializer(c.serializer),
		sqs.Session(c.awsSession),
	)
	if err != nil {
		return err
	}
	c.broker = brkr
	log.Debug("Submitting to queue=", c.JobQueueName())
	err = brkr.Publish(
		c.JobQueueName(),
		&broker.Message{
			ID: c.ID,
			Header: map[string]string{
				"id":              c.ID,
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

	fmt.Fprintln(c.options.stdout, color.GreenString("✱ Your job request has been posted to the queue."))

	c.spinner = spinner.New(spinner.CharSets[11], 100*time.Millisecond)
	c.spinner.Suffix = " Waiting for the server to process your request..."
	c.spinner.Writer = c.options.stdout
	c.spinner.Start()

	return nil
}

// Subscribe ...
func (c *client) Subscribe() error {
	redisConn, err := redis.New()
	if err != nil {
		return errors.Wrap(err, "cannot create a redis connection")
	}

	c.pubsubConn = redisConn

	subscribeChannel := config.App.Name + "/log-" + c.ID
	subscriber, err := redis.NewSubscriber(redisConn, subscribeChannel)
	if err != nil {
		return errors.Wrap(err, "cannot create redis subscriber")
	}

	c.resultHandler(subscriber.Start())

	c.subscribers = append(c.subscribers, subscriber)
	return nil
}

// Connect ...
func (c *client) Connect() error {
	if err := c.broker.Connect(); err != nil {
		return err
	}
	c.isConnected = true
	return nil
}

// Disconnect ...
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

// Wait ...
func (c *client) Wait() error {
	<-c.done
	return nil
}

func (c *client) authenticate(profilePath string) error {

	fmt.Fprintln(c.options.stdout, color.GreenString("✱ Checking your athentication credentials."))

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
