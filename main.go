package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"syscall"
	"time"

	"github.com/robfig/cron/v3"
)

type slackRequestBody struct {
	Text string `json:"text"`
}

type openlist struct {
	Islands []info `json:"islands"`
}

type info struct {
	Desc string `json:"description"`
}

// change this to yours
var webhookURL = "https://hooks.slack.com/services/example"

// example: var re = regexp.MustCompile(`(?i)cool pansy crown`)
var re = regexp.MustCompile(`(?i)crown|wreath`)

var turnipURL = "https://api.turnip.exchange/islands/"
var requestPayload = `{"islander":"neither","category":"crafting"}`

func main() {

	pollturnip()

	log.Print("Starting Cron Jobs")
	c := cron.New(cron.WithLogger(
		cron.VerbosePrintfLogger(log.New(os.Stdout, "cron: ", log.LstdFlags))))
	_, err := c.AddFunc("*/10 * * * *", pollturnip)
	if err != nil {
		panic(err)
	}

	c.Start()
	defer c.Stop()

	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		fmt.Println()
		fmt.Println(sig)
		done <- true
	}()

	fmt.Println("awaiting signal")
	<-done
	fmt.Println("exiting")

}

func pollturnip() {
	// Make the api call
	log.Print("polling")
	resp, err := http.Post(turnipURL, "application/json", bytes.NewBuffer([]byte(requestPayload)))
	if err != nil {
		return
	}

	// Read the body into a string
	openisland, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	// Unmarshal it into a struct
	decoded := openlist{}
	err = json.Unmarshal(openisland, &decoded)
	if err != nil {
		return
	}

	for _, il := range decoded.Islands {
		if re.MatchString(il.Desc) {
			log.Printf("found an island %s", il.Desc)
			err := sendSlackNotification(webhookURL, il.Desc)
			if err != nil {
				log.Printf("cannot send message to slack %v", err)
			}
		}
	}

}

// SendSlackNotification will post to an 'Incoming Webook' url setup in Slack Apps. It accepts
// some text and the slack channel is saved within Slack.
func sendSlackNotification(webhookURL string, msg string) error {

	slackBody, _ := json.Marshal(slackRequestBody{Text: msg})
	req, err := http.NewRequest(http.MethodPost, webhookURL, bytes.NewBuffer(slackBody))
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	if buf.String() != "ok" {
		return errors.New("Non-ok response returned from Slack")
	}
	return nil
}
