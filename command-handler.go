package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"sync/atomic"
	"time"

	jsoniter "github.com/json-iterator/go"

	"github.com/bwmarrin/discordgo"
	"github.com/robfig/cron"
)

type commandArgs struct {
	about    bool
	cronHelp bool
	run      bool
	nextRun  bool
	duration int
	limit    int
	cron     string
	loc      string
	channel  bool
	info     bool
}

const timeFormat = "15:04:05 02/01/2006 -0700"

var lockBot int32
var json = jsoniter.ConfigCompatibleWithStandardLibrary

func commandHandle(s *discordgo.Session, m *discordgo.MessageCreate, fls *flag.FlagSet, args []string, buf *bytes.Buffer) error {
	if atomic.CompareAndSwapInt32(&lockBot, 0, 1) == false {
		return errors.New("Bot is not available right now because of handling other command")
	}
	defer atomic.StoreInt32(&lockBot, 0)
	defer func() {
		// save settings
		b, err := json.Marshal(defJobArg)
		if err != nil {
			log.Println(err)
			return
		}
		err = ioutil.WriteFile(defJobArgJSONFile, b, 0644)
		if err == nil {
			log.Println("Saved state to:", defJobArgJSONFile)
		} else {
			log.Println(err)
		}
	}()
	cmdArg := &commandArgs{}
	fls.BoolVar(&cmdArg.about, "about", false, "Show about")
	fls.BoolVar(&cmdArg.info, "info", false, "Show current running state")
	fls.BoolVar(&cmdArg.channel, "set-channel", false, "Set current channel to remind")
	fls.BoolVar(&cmdArg.cronHelp, "cron-help", false, "Show cron help")
	fls.BoolVar(&cmdArg.run, "run", false, "Alias of -now")
	fls.BoolVar(&cmdArg.run, "show", false, "Alias of -now")
	fls.BoolVar(&cmdArg.run, "now", false, "Check events by now")
	fls.BoolVar(&cmdArg.nextRun, "next-run", false, "Get next run time")
	// duration between event start and end time
	fls.IntVar(&cmdArg.limit, "limit", defJobArg.Limit, "Set events limit")
	fls.IntVar(&cmdArg.duration, "duration", defJobArg.Duration, "Set duration in day(s)")
	fls.StringVar(&cmdArg.cron, "cron", defJobArg.Cron, "Set cron rule string")
	fls.StringVar(&cmdArg.loc, "timezone", defJobArg.LocStr, "Set time zone")
	err := fls.Parse(args[:])
	if err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}

	invalidOpt := true

	if cmdArg.run == false && cmdArg.channel == false && len(defJobArg.Channel) < 1 {
		fmt.Fprintln(buf, "Warning: no channel is provided, please set the channel to notify by option: -set-channel")
	}

	if cmdArg.limit != defJobArg.Limit {
		invalidOpt = false
		if cmdArg.limit < 1 {
			fmt.Fprintln(buf, "Invalid limit:", cmdArg.limit, "(want > 0)")
			return nil
		}
		if cmdArg.limit > 100 {
			fmt.Fprintln(buf, "Too much events:", cmdArg.limit, "(want <= 100)")
		}
		defJobArg.Limit = cmdArg.limit
		fmt.Fprintln(buf, "Set limit:", cmdArg.limit, "event(s)")
	}

	if cmdArg.duration != defJobArg.Duration {
		invalidOpt = false
		if cmdArg.duration < 1 {
			fmt.Fprintln(buf, "Invalid duration:", cmdArg.duration, "(want > 0)")
			return nil
		}
		defJobArg.Duration = cmdArg.duration
		fmt.Fprintln(buf, "Set duration:", cmdArg.duration, "day(s)")
	}

	if cmdArg.loc != defJobArg.LocStr {
		invalidOpt = false
		if len(cmdArg.loc) < 1 {
			return errors.New("Error: Empty timezone")
		}
		loc, err := time.LoadLocation(cmdArg.loc)
		if err != nil {
			return err
		}
		defJobArg.Loc = loc
		defJobArg.LocStr = cmdArg.loc
		err = makeCronJob()
		if err != nil {
			return err
		}
		fmt.Fprintln(buf, "Set timezone:", cmdArg.loc)
	}

	if cmdArg.cron != defJobArg.Cron {
		invalidOpt = false
		if len(cmdArg.cron) < 1 {
			return errors.New("Error: Empty cron rule")
		}
		schedule, err := cron.Parse(cmdArg.cron)
		if err != nil {
			return err
		}

		defJobArg.Cron = cmdArg.cron
		err = makeCronJob()
		if err != nil {
			return err
		}

		fmt.Fprintln(buf, "Set new cron rule:", cmdArg.cron)
		firstRun := schedule.Next(time.Now().In(defJobArg.Loc))
		fmt.Fprintln(buf, "First run:", firstRun.Format(timeFormat))
		secondRun := schedule.Next(firstRun)
		fmt.Fprintln(buf, "Second run:", secondRun.Format(timeFormat))
		fmt.Fprintln(buf, "Third run:", schedule.Next(secondRun).Format(timeFormat))
	}

	if cmdArg.run {
		invalidOpt = false
		err = getCTFTime2Discord(s, m)
		if err != nil {
			return err
		}
	}

	if cmdArg.nextRun {
		invalidOpt = false
		schedule, err := cron.Parse(defJobArg.Cron)
		if err != nil {
			return err
		}
		firstRun := schedule.Next(time.Now().In(defJobArg.Loc))
		fmt.Fprintln(buf, "Next run:", firstRun.Format(timeFormat))
		secondRun := schedule.Next(firstRun)
		fmt.Fprintln(buf, "Next(2) run:", secondRun.Format(timeFormat))
		fmt.Fprintln(buf, "Next(3) run:", schedule.Next(secondRun).Format(timeFormat))
	}

	if cmdArg.cronHelp {
		invalidOpt = false
		buf.WriteString("\nCron rule string descriptions:\n```\nField name   | Allowed values  | Allowed special characters\n----------   | --------------  | --------------------------\nSeconds      | 0-59            | * / , -\nMinutes      | 0-59            | * / , -\nHours        | 0-23            | * / , -\nDay of month | 1-31            | * / , - ?\nMonth        | 1-12 or JAN-DEC | * / , -\nDay of week  | 0-6 or SUN-SAT  | * / , - ?\n```\n")
	}

	if cmdArg.channel {
		invalidOpt = false
		defJobArg.Channel = m.ChannelID
		fmt.Fprintln(buf, `Set this channel to notify OK`)
		fmt.Fprintln(buf, `Current channel to notify:`, m.ChannelID)
	}

	if cmdArg.info {
		invalidOpt = false
		fmt.Fprintln(buf, "```md\n# Current running state:\n* Cron rule:", defJobArg.Cron)
		fmt.Fprintln(buf, `* Duration:`, defJobArg.Duration, "day(s)")
		fmt.Fprintln(buf, `* TimeZone:`, defJobArg.Loc.String())
		fmt.Fprintln(buf, `* Limit events:`, defJobArg.Limit, "event(s)")
		fmt.Fprintln(buf, `* Channel:`, defJobArg.Channel+"```")
	}

	if cmdArg.about {
		invalidOpt = false
		fmt.Fprintln(buf, `This bot automatically fetches ctf events on ctftime.org and then notifies a channel on discord.`)
		fmt.Fprintln(buf, `Author: @vinhjaxt`)
		fmt.Fprintln(buf, `Credits: @bwmarrin, @robfig, @valyala, @tidwall,..`)
	}

	// No options matched, print usage
	if invalidOpt {
		fls.Usage()
	}
	return nil
}
