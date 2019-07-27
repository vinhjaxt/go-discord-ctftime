package main

import (
	"bytes"
	"errors"
	"log"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/tidwall/gjson"

	"github.com/valyala/fasthttp"
)

var defaultHeaders = map[string]string{
	"Accept":                    "*/*",
	"Accept-Language":           "en-US;q=0.8,en;q=0.7",
	"DNT":                       "1",
	"Upgrade-Insecure-Requests": "1",
	"Pragma":                    "no-cache",
	"Cache-Control":             "no-cache",
	"Accept-Encoding":           "gzip, deflate",
	"User-Agent":                "Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/79.0.3578.80 Safari/537.36",
}

var CTFTimeLock int32

// use bot.ChannelMessageSend to send message to channel
func getCTFTime2Discord(s *discordgo.Session, m *discordgo.MessageCreate) error {
	if atomic.CompareAndSwapInt32(&CTFTimeLock, 0, 1) == false {
		return errors.New("Still fetching ctf events..")
	}
	defer atomic.StoreInt32(&CTFTimeLock, 0)
	var events []gjson.Result
	var err error
	channel := defJobArg.Channel
	if m != nil {
		channel = m.ChannelID
	}
	sender := s
	if sender == nil {
		sender = bot
	}
	_, err = sender.ChannelMessageSend(channel, "Fetching ctftime events..")
	if err != nil {
		log.Println(err)
	}
	retries := 0
	maxRetries := 9
	for {
		events, err = getCTFTime()
		if err == nil {
			break
		} else {
			if maxRetries != 0 && retries > maxRetries {
				log.Println("Get ctftime.org failed, max retries exceeded")
				return err
			}
			time.Sleep(2 * time.Second)
			retries++
			continue
		}
	}
	// events: [{"organizers": [{"id": 59759, "name": "redpwn"}], "onsite": false, "finish": "2019-08-16T16:00:00+00:00", "description": "A ctf hosted by redpwn", "weight": 0.00, "title": "RedpwnCTF 2019", "url": "https://ctf.redpwn.xyz/", "is_votable_now": false, "restrictions": "Open", "format": "Jeopardy", "start": "2019-08-12T16:00:00+00:00", "participants": 21, "ctftime_url": "https://ctftime.org/event/834/", "location": "", "live_feed": "", "public_votable": true, "duration": {"hours": 0, "days": 4}, "logo": "", "format_id": 1, "id": 834, "ctf_id": 331}]
	if len(events) < 1 {
		return errors.New("No ctf events")
	}
	contents := "## Upcomming CTF events\n"
	count := 0
	for _, event := range events {
		restriction := event.Get("restrictions").String()
		if len(restriction) > 0 && strings.ToLower(restriction) != "open" {
			continue
		}
		if event.Get("onsite").Bool() == true {
			continue
		}
		startTime, err := time.Parse(time.RFC3339, event.Get("start").String())
		if err != nil {
			log.Println(err)
			continue
		}
		startTime = startTime.In(defJobArg.Loc)
		duration := ""
		if event.Get("duration.days").Int() != 0 {
			duration += event.Get("duration.days").String() + " day(s)"
		}
		if event.Get("duration.hours").Int() != 0 {
			if len(duration) > 0 {
				duration += " + "
			}
			duration += event.Get("duration.hours").String() + " hour(s)"
		}
		endTime, err := time.Parse(time.RFC3339, event.Get("finish").String())
		if err != nil {
			log.Println(err)
			continue
		}
		endTime = endTime.In(defJobArg.Loc)
		count++
		markdown := "\n# " + event.Get("title").String() + ": " + event.Get("url").String()
		markdown += "\n* By: " + event.Get("organizers.0.name").String()
		markdown += "\n* Format: " + event.Get("format").String()
		markdown += "\n* Description: " + getExcerpt(event.Get("description").String(), 60)
		markdown += "\n* Duration: " + duration
		markdown += "\n* Start: " + startTime.Format(timeFormat)
		markdown += "\n* End: " + endTime.Format(timeFormat)
		markdown += "\n* CTFTime: " + event.Get("ctftime_url").String()
		if len(markdown)+len(contents) > 2000 {
			_, err = sender.ChannelMessageSend(channel, "```md\n"+contents+"```")
			if err != nil {
				log.Println(err)
			}
			contents = markdown
		} else {
			contents += markdown
		}

		if count >= defJobArg.Limit {
			break
		}
	}

	if count < 1 {
		contents = "No ctf events found."
	}
	_, err = sender.ChannelMessageSend(channel, "```md\n"+contents+"```")
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func getCTFTime() ([]gjson.Result, error) {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	for key, value := range defaultHeaders {
		req.Header.Set(key, value)
	}

	uri := fasthttp.AcquireURI()
	defer fasthttp.ReleaseURI(uri)
	uri.Update("https://ctftime.org/api/v1/events/?limit=100") // ?limit=100&start=1563995670&finish=1564600485
	queries := uri.QueryArgs()
	// queries.Set("limit", strconv.Itoa(defJobArg.limit+100))
	unixTimestamp := time.Now().Unix()
	queries.Set("start", strconv.FormatInt(unixTimestamp, 10))
	queries.Set("finish", strconv.FormatInt(unixTimestamp+int64(defJobArg.Duration)*86400, 10))

	req.SetRequestURI(uri.String())

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)
	err := fasthttp.DoTimeout(req, resp, 7*time.Second)
	if err != nil {
		return nil, err
	}
	body, err := GetResponseBody(resp)
	if err != nil {
		return nil, err
	}
	jsBody := gjson.ParseBytes(body)
	if jsBody.IsArray() == false {
		return nil, errors.New("No array found in response body")
	}
	return jsBody.Array(), nil
}

// GetResponseBody return plain response body of resp
func GetResponseBody(resp *fasthttp.Response) ([]byte, error) {
	var contentEncoding = resp.Header.Peek("Content-Encoding")
	if len(contentEncoding) < 1 {
		return resp.Body(), nil
	}
	if bytes.Equal(contentEncoding, []byte("gzip")) {
		return resp.BodyGunzip()
	}
	if bytes.Equal(contentEncoding, []byte("deflate")) {
		return resp.BodyInflate()
	}
	return nil, errors.New("unsupported response content encoding: " + string(contentEncoding))
}

func getExcerpt(str string, limit int) string {
	if len(str) > limit {
		return strings.Trim(str[0:limit], "\r\n\t ") + ".."
	}
	return str
}
