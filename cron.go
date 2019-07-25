package main

import (
	"errors"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/robfig/cron"
)

type jobArgs struct {
	Cron     string         `json:"cron-rule"`
	Duration int            `json:"duration"`
	LocStr   string         `json:"location-string"`
	Loc      *time.Location `json:"location"`
	Limit    int            `json:"limit"`
	Channel  string         `json:"channel"`
}

const defJobArgJSONFile = "./go-discord-ctftime.json"

var defJobArg = &jobArgs{
	Cron:     "0 0 0 * * 0",
	Duration: 7,
	LocStr:   "Asia/Ho_Chi_Minh",
	Limit:    7,
}
var cronJob *cron.Cron

func initJob() error {
	var err error
	if _, err = os.Stat(defJobArgJSONFile); err == nil {
		b, err := ioutil.ReadFile(defJobArgJSONFile)
		if err == nil {
			err = json.Unmarshal(b, defJobArg)
			if err == nil {
				log.Println("Restored state from:", defJobArgJSONFile)
			} else {
				log.Println(err)
			}
		} else {
			log.Println(err)
		}
	}
	defJobArg.Loc, err = time.LoadLocation(defJobArg.LocStr)
	if err != nil {
		return err
	}
	return makeCronJob()
}

func makeCronJob() error {
	if defJobArg.Loc == nil {
		return errors.New("No location found")
	}
	if cronJob != nil {
		cronJob.Stop()
	}
	cronJob = cron.NewWithLocation(defJobArg.Loc)
	log.Println("Make new CronJob:", defJobArg.Cron)
	err := cronJob.AddFunc(defJobArg.Cron, doJob)
	if err != nil {
		return err
	}
	cronJob.Start()
	return nil
}

func doJob() {
	if len(defJobArg.Channel) < 1 {
		log.Println("No channel set")
		return
	}
	err := getCTFTime2Discord(nil, nil)
	if err != nil {
		log.Println(err)
		if len(defJobArg.Channel) > 0 {
			_, err := bot.ChannelMessageSend(defJobArg.Channel, err.Error())
			if err != nil {
				log.Println(err)
			}
		}
	}
}
