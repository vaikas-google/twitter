package main

import (
	"fmt"
	"time"

	"github.com/dghubble/go-twitter/twitter"
)

type searcher struct {
	client    *twitter.Client
	query     string
	frequency int64 // in seconds
	handler   func(*twitter.Tweet) error
	stop      <-chan struct{}
	sinceID   int64 // keep track of what tweets we've seen so far
	stream    bool  // use streaming
}

func NewSearcher(client *twitter.Client, query string, frequency int64, handler func(*twitter.Tweet) error, stop <-chan struct{}, stream bool) *searcher {
	return &searcher{client: client, query: query, frequency: frequency, handler: handler, stop: stop, stream: stream}
}

func (s *searcher) run() {
	if s.stream {
		s.streamer()
	} else {
		s.restful()
	}
}

// restful method uses the restful API, aka polls.
func (s searcher) restful() {
	tickChan := time.NewTicker(5 * time.Second).C
	go func() {
		for {
			select {
			case <-tickChan:
				s.search()
			case <-s.stop:
				fmt.Println("Exiting")
			}
		}

	}()
}

// search uses REST api and polls it...
func (s *searcher) search() {
	search, resp, err := s.client.Search.Tweets(&twitter.SearchTweetParams{
		Query:           s.query,
		Lang:            "en",
		Count:           100,
		SinceID:         s.sinceID,
		IncludeEntities: twitter.Bool(true),
	})

	if err != nil {
		fmt.Printf("Error executing search %v\n", err)
		if resp != nil {
			fmt.Printf("Response Status: %s", resp.Status)
		}
		return
	}
	fmt.Printf("Got %d new tweets\n", len(search.Statuses))
	successes := 0
	for _, t := range search.Statuses {
		handlerErr := s.handler(&t)
		if handlerErr != nil {
			fmt.Printf("Failed to post: %s\n", err)
			break
		}
		successes = successes + 1
		if t.ID > s.sinceID {
			s.sinceID = t.ID
		}
	}
	fmt.Printf("Sent %d new tweets\n", successes)
}

func (s *searcher) streamer() {
	params := &twitter.StreamFilterParams{
		Track:         []string{s.query},
		StallWarnings: twitter.Bool(true),
	}
	demux := twitter.NewSwitchDemux()
	demux.Tweet = func(t *twitter.Tweet) {
		fmt.Printf("Got tweet from %s", t.User.Name)
		if handleErr := s.handler(t); handleErr != nil {
			fmt.Printf("Failed to post: %s\n", handleErr)
		}
	}

	stream, err := s.client.Streams.Filter(params)
	if err != nil {
		fmt.Printf("Failed to create filter: %s", err)
		return
	}
	go demux.HandleChan(stream.Messages)
}
