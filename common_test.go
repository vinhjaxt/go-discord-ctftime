package main

import (
	"bytes"
	"flag"
	"log"
	"testing"
	"time"

	"github.com/robfig/cron"
)

func Test(t *testing.T) {
	args, err := parseCommandLine(` ""      '2"        3'        4          5            `)
	if err != nil {
		log.Fatalln(err)
	}
	for _, arg := range args {
		log.Println(arg)
	}
}

func TestFlagSet(t *testing.T) {
	args, err := parseCommandLine(`-h "-now=13 2"      '2"        3'        -xxxx="4 5 6  7     x"                `)
	if err != nil {
		log.Fatalln(err)
	}
	fls := flag.NewFlagSet("command", flag.ContinueOnError)
	var buf bytes.Buffer
	fls.SetOutput(&buf)
	arg1 := fls.String("now", "", "Check events by now")
	err = fls.Parse(args[:])
	if err != nil {
		if err == flag.ErrHelp {
			log.Println(buf.String())
			buf.Reset()
		} else {
			log.Fatalln(err)
		}
	}
	fls.Usage()
	log.Println(buf.String())
	log.Println(*arg1, fls.Args())
}

func TestCron(t *testing.T) {
	loc, err := time.LoadLocation("UTC")
	if err != nil {
		log.Fatalln(err)
	}
	// newCron := cron.NewWithLocation()
	schedule, err := cron.Parse("0 * * * * *")
	if err != nil {
		log.Fatalln(err)
	}

	firstRun := schedule.Next(time.Now().In(loc))
	log.Println(firstRun)
	log.Println(schedule.Next(firstRun))
}

func TestGetCTFTime2Discord(t *testing.T) {
	err := getCTFTime2Discord(nil, nil)
	if err != nil {
		log.Fatalln(err)
	}
}
