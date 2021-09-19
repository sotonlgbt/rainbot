package main

import (
	"time"
)

type Config struct {
	// maps guild IDs to configs
	Guilds map[uint64]GuildConfig
}

type GuildConfig struct {
	AlumniGuild bool `yaml:"alumniGuild"`
	// maps channel IDs to configs
	Channels map[uint64]ChannelConfig
}

type ChannelConfig struct {
	ReapDuration time.Duration `yaml:"reapDuration"`
}
