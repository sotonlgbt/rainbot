package main

import (
	"time"

	"github.com/diamondburned/arikawa/v3/discord"
)

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
