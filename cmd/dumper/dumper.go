/*
Copyright 2018 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"fmt"
	"log"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/dghubble/go-twitter/twitter"
)

func myFunc(event cloudevents.Event) error {
	log.Printf("Received Cloud Event Context as: %+v", event.Context)
	var tweet twitter.Tweet
	if err := event.DataAs(&tweet); err != nil {
		return fmt.Errorf("Unable to unpack tweet: %w", err)
	}
	log.Printf("Got tweet from %q text: %q", tweet.User.Name, tweet.Text)

	return nil
}

func main() {
	c, err := cloudevents.NewDefaultClient()
	if err != nil {
		log.Fatalf("Unable to initialize client: %s", err)
	}
	log.Fatal(c.StartReceiver(context.Background(), myFunc))
}
