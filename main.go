package main

import (
	"context"
	"log"
	"os"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/state"
)

// Configuration and flags are set up in config.go!

func main() {
	var token = os.Getenv("BOT_TOKEN")
	if token == "" {
		log.Fatalln("No $BOT_TOKEN given.")
	}

	appID, err := discord.ParseSnowflake(os.Getenv("APP_ID"))
	if err != nil {
		log.Fatalf("Invalid snowflake for $APP_ID: %v", err)
	}

	s, err := state.New("Bot " + token)
	if err != nil {
		log.Fatalln("Session failed:", err)
	}

	bot := Bot{State: s}

	switch {
	case reaperMode:
		log.Println("Reaper mode active")

		for guildID, guildConfig := range config.Guilds {
			for channelID, channelConfig := range guildConfig.Channels {
				log.Println("Reaping channel", channelID, "from guild", guildID)
				err = bot.ReapChannelMessages(discord.ChannelID(channelID), channelConfig.ReapDuration)
				if err != nil {
					log.Fatalln("Failed reaping channel", channelID, "from guild", guildID, "with error", err)
				}
			}
		}

		log.Println("Reaping done, ending")
	case warnInvalidMode:
		log.Println("Invalid user warning mode active")
		if warnInvalidDryRunMode {
			log.Println("Dry run active - no real messages will be sent!")
		}

		for guildID, guildConfig := range config.Guilds {
			// For now, we ignore alumni guilds in here
			if guildConfig.AlumniGuild {
				continue
			}

			bot.warnInvalidUsers(guildID)
		}

		log.Println("Invalid user warning done, ending")
	case purgeInvalidMode:
		log.Println("Invalid user purging mode active")

		for guildID, guildConfig := range config.Guilds {
			// For now, we ignore alumni guilds in here
			if guildConfig.AlumniGuild {
				continue
			}

			bot.PurgeInvalidUsers(guildID)
		}

		log.Println("Invalid user purging done, ending")
	default:
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
}
