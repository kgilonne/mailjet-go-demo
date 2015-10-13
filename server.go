package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	. "github.com/mailjet/mailjet-apiv3-go"
	"github.com/mailjet/mailjet-apiv3-go/resources"
	"log"
	"net/http"
	"os"
)

type Config struct {
	MailjetConfig struct {
		APIKey    string
		APISecret string
		Email     string
		Domain    string
	}
	SlackConfig struct {
		Token   string
		Channel string
		Emoji   string
	}
}

type Mailjet struct {
	Sender    string
	Recipient string
	Date      string
	From      string
	Subject   string
	Headers   map[string]string
	Parts     []struct {
		Headers    map[string]string
		ContentRef string
	}
	TextPart          string `json:"Text-part"`
	HtmlPart          string `json:"Html-part"`
	SpamAssassinScore float64
	CustomID          string
	Payload           string
}

type Slack struct {
	Channel    string `json:"channel"`
	Username   string `json:"username"`
	Text       string `json:"text"`
	Icon_emoji string `json:"icon_emoji"`
}

const slackWebhookBaseURL string = "https://hooks.slack.com/services/"

var configFile = flag.String("f", "config.json", "configuration file")
var port = flag.Int("p", 3000, "port of the server")

var Usage = func() {
	flag.PrintDefaults()
}

var config Config

func Webhook(w http.ResponseWriter, r *http.Request) {
	// Decode request body
	var mj Mailjet
	err := json.NewDecoder(r.Body).Decode(&mj)
	if err != nil {
		fmt.Fprintln(w, err)
		return
	}

	// Encode body for new request
	slackConfig := config.SlackConfig
	slack := Slack{
		Channel:    slackConfig.Channel,
		Username:   mj.From,
		Text:       mj.TextPart,
		Icon_emoji: slackConfig.Emoji,
	}
	body, err := json.Marshal(slack)
	if err != nil {
		fmt.Fprintln(w, err)
		return
	}

	// Create new request for slack api
	req, err := http.NewRequest("POST", slackWebhookBaseURL+slackConfig.Token, bytes.NewBuffer(body))
	if err != nil {
		fmt.Fprintln(w, "Error creating request:", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	// Do new request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintln(w, "Slack error response:", err)
		return
	}
	fmt.Fprintln(w, resp.Status)
}

func createParseRoute(mj *MailjetClient, email, url string) {
	var res []resources.Parseroute
	fmr := &FullMailjetRequest{
		Info:    &MailjetRequest{Resource: "parseroute"},
		Payload: resources.Parseroute{URL: url + "/webhook", Email: email},
	}
	err := mj.Post(fmr, &res)
	if err != nil {
		log.Println("Error creating new instance of the parse API: ", err)
	} else {
		log.Println("Parse route email: ", res[0].Email)
	}
}

func checkParseRoute(mj *MailjetClient, email, url string) {
	var res []resources.Parseroute
	info := &MailjetRequest{Resource: "parseroute", AltID: email}
	err := mj.Get(info, &res)
	if err != nil {
		log.Println("Error getting instance of the parse API: ", err)
		createParseRoute(mj, email, url)
	} else {
		log.Println("Email already used: ", res[0].Email)
	}
}

func main() {
	flag.Usage = Usage
	flag.Parse()

	r, err := os.Open(*configFile)
	if err != nil {
	}
	err = json.NewDecoder(r).Decode(&config)
	if err != nil {
		log.Fatal(fmt.Sprintf("Unable to read the config file (%s): %s", *configFile, err))
	}
	log.Println(fmt.Sprintf("Read config %s: %+v", *configFile, config))

	mjConfig := config.MailjetConfig
	mj := NewMailjetClient(mjConfig.APIKey, mjConfig.APISecret)

	url := fmt.Sprintf("http://%s:%d", mjConfig.Domain, *port)

	go checkParseRoute(mj, mjConfig.Email, url)

	log.Println("Server started: ", url)
	http.HandleFunc("/webhook", Webhook)
	log.Fatal(http.ListenAndServe(fmt.Sprintf("%s:%d", mjConfig.Domain, *port), nil))
}
