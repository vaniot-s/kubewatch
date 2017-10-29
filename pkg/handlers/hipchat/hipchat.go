/*
Copyright 2016 Skippbox, Ltd.

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

package hipchat

import (
	"fmt"
	"log"
	"os"

	hipchat "github.com/tbruyelle/hipchat-go/hipchat"

	"github.com/skippbox/kubewatch/config"
	"github.com/skippbox/kubewatch/pkg/event"
	kbEvent "github.com/skippbox/kubewatch/pkg/event"
	"net/url"
)

var hipchatColors = map[string]hipchat.Color{
	"Normal":  hipchat.ColorGreen,
	"Warning": hipchat.ColorYellow,
	"Danger":  hipchat.ColorRed,
}

var hipchatErrMsg = `
%s

You need to set both hipchat token and channel for hipchat notify,
using "--token/-t" and "--channel/-c", or using environment variables:

export KW_HIPCHAT_TOKEN=hipchat_token
export KW_HIPCHAT_CHANNEL=hipchat_channel

Command line flags will override environment variables

`

// Hipchat handler implements handler.Handler interface,
// Notify event to hipchat room
type Hipchat struct {
	Token   string
	Room 	string
	Url     string
}

// Init prepares hipchat configuration
func (s *Hipchat) Init(c *config.Config) error {
	url	:= c.Handler.Hipchat.Url
	room := c.Handler.Hipchat.Room
	token := c.Handler.Hipchat.Token

	if token == "" {
		token = os.Getenv("KW_HIPCHAT_TOKEN")
	}

	if room == "" {
		room = os.Getenv("KW_HIPCHAT_ROOM")
	}

	if url == "" {
		url = os.Getenv("KW_HIPCHAT_URL")
	}

	s.Token = token
	s.Room  = room
	s.Url   = url

	return checkMissingHipchatVars(s)
}

func (s *Hipchat) ObjectCreated(obj interface{}) {
	notifyHipchat(s, obj, "created")
}

func (s *Hipchat) ObjectDeleted(obj interface{}) {
	notifyHipchat(s, obj, "deleted")
}

func (s *Hipchat) ObjectUpdated(oldObj, newObj interface{}) {
	notifyHipchat(s, newObj, "updated")
}

func notifyHipchat(s *Hipchat, obj interface{}, action string) {
	e := kbEvent.New(obj, action)

	client := hipchat.NewClient(s.Token)
	if s.Url != "" {
		baseUrl, err := url.Parse(s.Url)
		if err != nil {
			panic(err)
		}
		client.BaseURL = baseUrl
	}

	notificationRequest := prepareHipchatNotification(e)
	resp, err := client.Room.Notification(s.Room, &notificationRequest)

	if err != nil {
		log.Printf("%s\n", err)
		return
	}

	log.Printf("Message successfully sent to room %s with resp %s", s.Room, resp)
}

func checkMissingHipchatVars(s *Hipchat) error {
	if s.Token == "" || s.Room == "" {
		return fmt.Errorf(hipchatErrMsg, "Missing hipchat token or room")
	}

	return nil
}

func prepareHipchatNotification(e event.Event) hipchat.NotificationRequest {
	msg := fmt.Sprintf(
		"A %s in namespace %s has been %s: %s",
		e.Kind,
		e.Namespace,
		e.Reason,
		e.Name,
	)


	notification := hipchat.NotificationRequest{
		Message: msg,
		Notify: true,
		From: "kubewatch",
	}

	if color, ok := hipchatColors[e.Status]; ok {

		notification.Color = color
	}

	return notification
}
