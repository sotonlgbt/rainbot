package main

import (
	"time"

	"github.com/diamondburned/arikawa/v3/discord"
)

type Config struct {
	// maps guild IDs to configs
	Guilds   map[discord.GuildID]GuildConfig
	Pronouns []string
}

type GuildConfig struct {
	AlumniGuild bool `yaml:"alumniGuild"`
	// maps channel IDs to configs
	Channels map[discord.GuildID]ChannelConfig
	Colours  []string
	Roles    []string
}

type ChannelConfig struct {
	ReapDuration time.Duration `yaml:"reapDuration"`
}
