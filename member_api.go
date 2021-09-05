package main

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"

	"github.com/diamondburned/arikawa/discord"
)

// getDiscordAuthLink returns the Discord authentication link
// from the authentication server. This relies on having a symlink
// to the authentication server's artisan install.
func getDiscordAuthLink(user discord.User) string {
	generatorCommand := exec.Command("php", fmt.Sprintf("artisan gayauth:generateDiscordAuthUrl %d", user.ID))
	var out bytes.Buffer
	generatorCommand.Stdout = &out
	err := generatorCommand.Run()
	if err != nil {
		log.Fatal(err)
	}
	return out.String()
}

// isDiscordAuthenticated checks whether a user is authenticated
// in the database for Discord, and returns true if they are, or
// false otherwise.
// TODO implement
func isDiscordAuthenticated(user discord.User) bool {
	return true
}
