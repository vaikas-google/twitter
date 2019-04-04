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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/cloudevents/sdk-go/pkg/cloudevents"
	"github.com/cloudevents/sdk-go/pkg/cloudevents/client"
	"github.com/dghubble/go-twitter/twitter"
	"github.com/kelseyhightower/envconfig"
)

type EnvConfig struct {
	Target string `required:"true"`
}

type poster struct {
	target string
}

type slackMessage struct {
	Text string `json:"text"`
}

//func (p *poster) send(ctx context.Context, tweet *twitter.Tweet) error {
func (p *poster) send(ctx context.Context, event cloudevents.Event) error {
	fmt.Printf("Got Event Context: %+v\n", event.Context)
	var tweet twitter.Tweet

	if err := event.DataAs(&tweet); err != nil {
		fmt.Printf("Got Data Error: %s\n", err.Error())
		return err
	}

	msg := &slackMessage{fmt.Sprintf("GOT Tweet from %q text: %q", tweet.User.Name, tweet.Text)}

	b, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error marshaling: %s", err)
		return err
	}

	req, err := http.NewRequest("POST", p.target, bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error posting: %s", err)
		return err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return errors.New(fmt.Sprintf("Failed to post to slack %q : %s", resp.Status, body))
	}
	return nil
}

func main() {
	var s EnvConfig
	err := envconfig.Process("slacker", &s)
	if err != nil {
		log.Fatal(err.Error())
	}

	p := poster{s.Target}

	c, err := client.NewDefault()
	if err != nil {
		log.Fatalf("failed to create client, %v", err)
	}
	log.Fatal(c.StartReceiver(context.Background(), p.send))
}
