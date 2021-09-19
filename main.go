package main

import (
	"context"
	"flag"
	"io/ioutil"
	"log"
	"os"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/joho/godotenv"
	"gopkg.in/yaml.v2"
)

// config stores the bot configuration.
var config Config = Config{}

// reaperMode is true when the bot is being launched to delete old messages.
var reaperMode bool

func init() {
	flag.BoolVar(&reaperMode, "reaperMode", false, "Sets the bot to be in reaper mode.")

	err := godotenv.Load(".env")

	if err != nil {
		log.Fatal("Error loading .env file")
	}

	flag.Parse()
}

func main() {
	var token = os.Getenv("BOT_TOKEN")
	if token == "" {
		log.Fatalln("No $BOT_TOKEN given.")
	}

	appID, err := discord.ParseSnowflake(os.Getenv("APP_ID"))
	if err != nil {
		log.Fatalf("Invalid snowflake for $APP_ID: %v", err)
	}

	configfile, err := ioutil.ReadFile("config.yml") // the file is inside the local directory
	if err != nil {
		log.Fatalln("Failed loading config file:", err)
	}

	err = yaml.Unmarshal(configfile, &config)
	if err != nil {
		log.Fatalln("Failed parsing config file:", err)
	}

	log.Println(config)

	s, err := state.New("Bot " + token)
	if err != nil {
		log.Fatalln("Session failed:", err)
	}

	if reaperMode {
		log.Println("Reaper mode active")

		for guildID, guildConfig := range config.Guilds {
			for channelID, channelConfig := range guildConfig.Channels {
				log.Println("Reaping channel", channelID, "from guild", guildID)
				err = ReapChannelMessages(discord.ChannelID(channelID), channelConfig.ReapDuration, s)
				if err != nil {
					log.Fatalln("Failed reaping channel", channelID, "from guild", guildID, "with error", err)
				}
			}
		}

		log.Println("Reaping done, ending")
		os.Exit(0)
	}

	bot := Bot{State: s}
	dispatcher := Dispatcher{Bot: bot}

	s.AddHandler(dispatcher.InteractionEventDispatcher)
	s.AddHandler(dispatcher.NewGuildMemberEventDispatcher)

	newCommands := []api.CreateCommandData{
		{
			Name:        "verification_button",
			Description: "Inserts a verification button in the current channel - for server owners only!",
		},
		{
			Name:        "pronoun_picker",
			Description: "Inserts a pronoun picker in the current channel - for server owners only!",
		},
		{
			Name:        "colour_picker",
			Description: "Inserts a colour picker in the current channel - for server owners only!",
		},
		{
			Name:        "role_picker",
			Description: "Inserts a general role picker in the current channel - for server owners only!",
		},
	}

	for _, command := range newCommands {
		_, err := s.CreateCommand(discord.AppID(appID), command)
		if err != nil {
			log.Fatalln("failed to create command:", err)
		}
	}

	s.AddIntents(gateway.IntentGuildMessages)
	s.AddIntents(gateway.IntentGuildMembers)
	s.AddIntents(gateway.IntentGuildInvites)
	s.AddIntents(gateway.IntentDirectMessages)

	if err := s.Open(context.Background()); err != nil {
		log.Fatalln("Failed to connect:", err)
	}
	defer s.Close()

	log.Println("Bot started")

	// Block forever.
	select {}
}
