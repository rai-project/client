package client

import (
	"context"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"path/filepath"

	"fmt"

	"github.com/AlekSi/pointer"
	"github.com/Unknwon/com"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	colorable "github.com/mattn/go-colorable"
	gamp "github.com/olebedev/go-gamp"
	"github.com/olebedev/go-gamp/client/gampops"
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
	"gopkg.in/yaml.v2"
)

type client struct {
	ID                    string
	uploadKey             string
	analyticsClient       *gampops.Client
	analyticsClientParams *gampops.CollectParams
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
	ranking               *model.Fa2017Ece408Job
	done                  chan bool
}

var (
	m2Name     = "_fixtures/m2.yml"
	m3Name     = "_fixtures/m3.yml"
	finalName  = "_fixtures/final.yml"
	evalName   = "_fixtures/eval.yml"
	m2Build    = FSMustByte(false, "/"+m2Name)
	m3Build    = FSMustByte(false, "/"+m3Name)
	finalBuild = FSMustByte(false, "/"+finalName)
	evalBuild  = FSMustByte(false, "/"+evalName)
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
	SubmissionUploadExpiration = func() time.Time {
		return time.Now().AddDate(0, 6, 0) // six months from now
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
		ctx:               context.Background(),
		directory:         "",
		isSubmission:      false,
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

	clnt := &client{
		ID:                  uuid.NewV4(),
		isConnected:         false,
		options:             options,
		serializer:          json.New(),
		configJobQueueName:  Config.JobQueueName,
		optionsJobQueueName: options.jobQueueName,
		done:                make(chan bool),
	}

	if Config.AnalyticsKey != "" {
		analyticsClient := gamp.New(options.ctx, Config.AnalyticsKey)
		clnt.analyticsClient = analyticsClient
	}

	return clnt, nil
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

func (c *client) RecordRanking() error {

	if c.ranking == nil {
		return errors.New("ranking uninitialized")
	}

	c.ranking.CreatedAt = time.Now()
	c.ranking.IsSubmission = c.options.isSubmission
	if c.options.submissionKind != custom {
		c.ranking.SubmissionTag = string(c.options.submissionKind)
	} else {
		c.ranking.SubmissionTag = c.options.customSubmissionTag
	}

	prof, err := provider.New()
	user := prof.Info()
	c.ranking.Username = user.Username
	c.ranking.Teamname = user.Team.Name
	log.Debug("Submission username: " + c.ranking.Username)
	log.Debug("Submission teamname: " + c.ranking.Teamname)

	if c.ranking.IsSubmission {
		if c.ranking.Teamname == "" {
			return errors.New("no team name found")
		}
	}

	log.Info("Connecting to submission database: ", config.App.Name)
	db, err := mongodb.NewDatabase(config.App.Name)
	if err != nil {
		return err
	}
	defer db.Close()

	log.Info("Connecting to table: rankings")
	col, err := model.NewFa2017Ece408JobCollection(db)
	if err != nil {
		return err
	}
	defer col.Close()

	err = col.Insert(c.ranking)
	log.Info("Inserted ranking")
	return err
}

// Validate ...
func (c *client) Validate() error {
	options := c.options

	// Authenticate user
	if err := c.authenticate(options.profilePath); err != nil {
		return err
	}

	if c.analyticsClient != nil {
		prof := c.profile.Info()
		teamName := ""
		if prof.User.Team != nil {
			teamName = prof.User.Team.Name
		}
		params := gampops.NewCollectParams().
			WithCid(pointer.ToString(prof.User.AccessKey)).
			WithTi(pointer.ToString(c.ID)).
			WithAn(pointer.ToString(config.App.Name)).
			WithAid(pointer.ToString(config.App.Version.Version)).
			WithAv(pointer.ToString(config.App.Version.BuildDate)).
			WithCid(pointer.ToString(prof.User.AccessKey)).
			WithTa(pointer.ToString(teamName)).
			WithUID(pointer.ToString(prof.User.Username)).
			WithT("event").
			WithEc(pointer.ToString(config.App.Version.Version)).
			WithEa(pointer.ToString("build")).
			WithEl(pointer.ToString("command")).
			WithSc(pointer.ToString("start"))
		if options.isSubmission {
			params = params.WithEa(pointer.ToString("submission")).
				WithEl(pointer.ToString(string(options.submissionKind)))
		}
		err := c.analyticsClient.Collect(params)
		if err != nil {
			log.WithError(err).Info("failed to publish analytics")
		} else {
			c.analyticsClientParams = params
		}
	}

	var buf []byte

	if options.isSubmission {
		switch options.submissionKind {
		case final:
			{
				buf = finalBuild
			}
		case m2:
			{
				buf = m2Build
			}
		case m3:
			{
				buf = m3Build
			}
		case custom:
			{
				log.Info("Using embedded eval build for custom submission")
				buf = evalBuild
			}
		default:
			{
				return errors.New("unrecognized submission type " + string(options.submissionKind))
			}
		}
		fmt.Fprintf(c.options.stdout, color.YellowString("Using the following build file for submission:\n%s"), string(buf))
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
			c.ranking = &model.Fa2017Ece408Job{}
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
	uploadExpiration := DefaultUploadExpiration()
	if c.options.isSubmission {
		uploadExpiration = SubmissionUploadExpiration()
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
func (c *client) Publish() error {

	profile := c.profile.Info()

	jobRequest := model.JobRequest{
		Base: model.Base{
			ID:        c.ID,
			CreatedAt: time.Now(),
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

	if c.analyticsClient != nil && c.analyticsClientParams != nil {
		c.analyticsClient.Collect(c.analyticsClientParams.WithSc(pointer.ToString("end")))
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
