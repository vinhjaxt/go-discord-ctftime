package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

// Chat @bot to usage bot
var prefix = "<@me>"

var botToken string
var bot *discordgo.Session

// use bot.ChannelMessageSend to send message to channel

func init() {
	if len(os.Args) > 1 {
		botToken = os.Args[1]
	}
}

func main() {
	if len(botToken) < 1 {
		log.Println("Usage:", os.Args[0], "[discord bot token]")
		return
	}
	var err error
	bot, err = discordgo.New("Bot " + botToken)
	if err != nil {
		log.Println("Error creating Discord session:", err)
		return
	}

	if prefix == "<@me>" {
		currentUser, err := bot.User("@me")
		if err != nil {
			log.Panicln(err)
		}
		prefix = currentUser.Mention()
	}

	// Register the messageCreate func as a callback for MessageCreate events.
	bot.AddHandler(messageCreate)

	// Open a websocket connection to Discord and begin listening. (auto reconnect)
	err = bot.Open()
	if err != nil {
		log.Println("Error opening connection:", err)
		return
	}

	err = initJob()
	if err != nil {
		log.Panicln(err)
	}

	// Wait here until CTRL-C or other term signal is received.
	log.Println("Bot is now running. Press CTRL-C to exit.")
	log.Println("Usage: Chat @bot -h for more information.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
	log.Println("Terminating..")
	// Cleanly close down the Discord session.
	bot.Close()
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the autenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}
	content := strings.Trim(m.Content, "\r\n\t ")
	if !strings.HasPrefix(content, prefix) {
		return
	}
	buf := new(bytes.Buffer)
	defer func() {
		output := buf.String()
		if len(output) < 1 {
			return
		}
		_, err := s.ChannelMessageSend(m.ChannelID, output)
		if err != nil {
			log.Println(err)
		}
	}()
	args, err := parseCommandLine(content[len(prefix):])
	if err != nil {
		fmt.Fprintln(buf, err)
		return
	}

	fls := flag.NewFlagSet(prefix, flag.ContinueOnError)
	fls.SetOutput(buf)

	err = commandHandle(s, m, fls, args, buf)
	if err != nil {
		fmt.Fprintln(buf, err)
	}
}
