package main

import (
	"context"
	"flag"
	"fmt"
	"strconv"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	"github.com/kelseyhightower/envconfig"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	"knative.dev/pkg/signals"
)

var (
	query  string
	stream bool
)

type EnvConfig struct {
	ConsumerKey       string `split_words:"true" required:"true"`
	ConsumerSecretKey string `split_words:"true" required:"true"`
	AccessToken       string `split_words:"true" required:"true"`
	AccessSecret      string `split_words:"true" required:"true"`

	Sink string `envconfig:"K_SINK" required:"true"`
}

func init() {
	flag.StringVar(&query, "query", "", "query string to look for")
	flag.BoolVar(&stream, "stream", true, "Use the streaming API instead of REST")
}

func main() {
	flag.Parse()

	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()

	logConfig := zap.NewProductionConfig()
	logConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	logger, err := logConfig.Build()
	if err != nil {
		panic(fmt.Sprintf("failed to create logger: %s", err))
	}

	var s EnvConfig
	err = envconfig.Process("twitter", &s)
	if err != nil {
		logger.Fatal(err.Error())
	}

	if query == "" {
		logger.Error("Need to specify query string")
		return
	}
	if s.Sink == "" {
		logger.Error("Need to specify sink")
		return
	}

	logger.Info("Starting and publishing to sink", zap.String("sink", s.Sink))
	logger.Info("querying for ", zap.String("query", query))
	logger.Info("streaming on ", zap.Bool("stream", stream))

	ceClient, err := cloudevents.NewDefaultClient()
	if err != nil {
		logger.Error("Failed to initialize CloudEvents client: %s", zap.Error(err))
		return
	}
	/*NewClient(sink, cloudevents.Builder{
		EventType: "com.twitter",
		Source:    "com.twitter",
	})*/

	publisher := publisher{ceClient: ceClient, logger: logger, target: s.Sink}

	config := oauth1.NewConfig(s.ConsumerKey, s.ConsumerSecretKey)
	token := oauth1.NewToken(s.AccessToken, s.AccessSecret)
	httpClient := config.Client(oauth1.NoContext, token)

	// Twitter client
	client := twitter.NewClient(httpClient)

	searcher := NewSearcher(client, logger, query, 5, publisher.postMessage, stopCh, stream)
	searcher.run()
	<-stopCh
}

type publisher struct {
	ceClient cloudevents.Client
	target   string
	logger   *zap.Logger
}

type simpleTweet struct {
	user string `json:"user"`
	text string `json:"text"`
}

func (p *publisher) postMessage(tweet *twitter.Tweet) error {
	eventTime, err := time.Parse(time.RubyDate, tweet.CreatedAt)

	if err != nil {
		p.logger.Info("Failed to parse created at: ", zap.Error(err))
		eventTime = time.Now()
	}
	event := cloudevents.NewEvent()
	event.SetSource("https://twitter.com/")
	event.SetType("com.twitter.tweet")
	event.SetData("application/json", tweet)
	event.SetTime(eventTime)
	event.SetID(strconv.FormatInt(tweet.ID, 10))

	// TODO: plumb a shared context between tweet receipt and this send.
	ctx := cloudevents.ContextWithTarget(context.Background(), p.target)

	p.logger.Info("Attempting to send to", zap.String("target", p.target), zap.String("event", event.ID()))

	if result := p.ceClient.Send(ctx, event); cloudevents.IsUndelivered(result) {
		return result
	}
	return nil
	/*tweet, cloudevents.V01EventContext{
		EventID:   strconv.FormatInt(tweet.ID, 10),
		EventTime: eventTime,
	})*/
}
