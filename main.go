/*
  at 10:00, send quotes via Twilio based on line number and date,
  as not to repeat any quotes until the end of the list has been reached.

  This is not meant to be idiomatic code or an example of prowess. This is,
  in essence, a quick-and-dirty script.
*/
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

var (
	quotes []string

	sid   string
	token string
	tURL  string

	recipients []string
	sender     string
)

func init() {
	data, err := ioutil.ReadFile("aaw.json")
	if err != nil {
		panic(err)
	}
	if err = json.Unmarshal(data, &quotes); err != nil {
		panic(err)
	}
	if len(quotes) == 0 {
		panic(errors.New("no quotes provided in aaw.json"))
	}
	sid = os.Getenv("TWILIO_SID")
	token = os.Getenv("TWILIO_TOKEN")
	tURL = "https://api.twilio.com/2010-04-01/Accounts/" + sid + "/Messages.json"
	sender = os.Getenv("TWILIO_SENDER")
	recipients = strings.Split(os.Getenv("TWILIO_RECIPIENTS"), ",")
	if len(sid) == 0 || len(token) == 0 || len(sender) == 0 || len(recipients) == 0 {
		panic(errors.New("invalid env vars"))
	}
}

func main() {
	fmt.Println("waiting to send quotes...")
	for {
		now := time.Now()
		y, m, d := now.Date()
		daysSinceNewYear := (int(m) * 31) + d                             // very rough calculation
		dailyQuoteNumber := abs(daysSinceNewYear%len(quotes) - 1)         // 0-based index
		time.Sleep(time.Until(time.Date(y, m, d, 10, 0, 0, 0, time.UTC))) // wait until 10:00
		if err := sendSMS(quotes[dailyQuoteNumber]); err != nil {
			fmt.Println(err)
		}
		time.Sleep(time.Hour) // forced sleep of minimum 1h (time to ^C block loop)
	}
}

func sendSMS(s string) error {
	s = fmt.Sprintf("%q\n    - Sun Tzu", s)
	for _, to := range recipients {
		msg := url.Values{}
		msg.Set("To", to)
		msg.Set("From", sender)
		msg.Set("Body", s)
		msgReader := *strings.NewReader(msg.Encode())
		c := &http.Client{}
		req, err := http.NewRequest("POST", tURL, &msgReader)
		if err != nil {
			return err
		}
		req.SetBasicAuth(sid, token)
		req.Header.Add("Accept", "application/json")
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		resp, err := c.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
			fmt.Println(resp.Status)
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			fmt.Println(string(body))
			return errors.New("Unexpected status code in Twilio response")
		}
		fmt.Println(s) // print the quote just because
	}
	return nil
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
