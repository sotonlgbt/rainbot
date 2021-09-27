package main

import (
	"flag"
	"io/ioutil"
	"log"
	"time"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/joho/godotenv"
	"gopkg.in/yaml.v2"
)

// config stores the bot configuration.
var config Config = Config{}

// reaperMode is true when the bot is being launched to delete old messages.
var reaperMode bool

// warnInvalidMode is true when the bot is being launched to warn users who are invalid of their pending removal.
var warnInvalidMode bool

// warnInvalidDryRunMode is true when the bot is in warnInvalid mode, but should not send any messages - just print the usernames to the console.
var warnInvalidDryRunMode bool

// warnInvalidDeadline sets a time when members should expect to be removed in an invalid member warning.
var warnInvalidDeadline string

// purgeInvalidMode is true when the bot is being launched to remove unsuitably-verified users.
var purgeInvalidMode bool

// init sets up our command line flags, loads our env file, and loads our config file.
func init() {
	flag.BoolVar(&reaperMode, "reaperMode", false, "Sets the bot to be in reaper mode.")
	flag.BoolVar(&warnInvalidMode, "warnInvalid", false, "Sets the bot to be in 'invalid user' warning mode.")
	flag.BoolVar(&warnInvalidDryRunMode, "warnInvalidDryRun", false, "Sets the bot to not message invalid users, but just print names to the console.")
	flag.StringVar(&warnInvalidDeadline, "warnInvalidDeadline", "a few days", "Sets a string to use as a timeframe for members to expect to be removed.")
	flag.BoolVar(&purgeInvalidMode, "purgeInvalid", false, "Sets the bot to be in 'invalid user' purging mode.")

	err := godotenv.Load(".env")

	if err != nil {
		log.Fatal("Error loading .env file")
	}

	flag.Parse()

	configfile, err := ioutil.ReadFile("config.yml") // the file is inside the local directory
	if err != nil {
		log.Fatalln("Failed loading config file:", err)
	}

	err = yaml.Unmarshal(configfile, &config)
	if err != nil {
		log.Fatalln("Failed parsing config file:", err)
	}

	// log.Println(config)
}

// Config holds the overall application configuration.
type Config struct {
	// maps guild IDs to configs
	Guilds   map[discord.GuildID]GuildConfig
	Pronouns []string
}

// GuildConfig holds configuration for a specific guild.
type GuildConfig struct {
	AlumniGuild bool `yaml:"alumniGuild"`
	// maps channel IDs to configs
	Channels map[discord.GuildID]ChannelConfig
	Colours  []string
	Roles    []string
}

// ChannelConfig holds configuration for a specific channel in a guild.
type ChannelConfig struct {
	ReapDuration time.Duration `yaml:"reapDuration"`
}
