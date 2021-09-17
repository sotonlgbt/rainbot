package main

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/diamondburned/arikawa/discord"
)

// getDiscordAuthLink returns the Discord authentication link
// from the authentication server. This relies on having a symlink
// to the authentication server's artisan install.
func getDiscordAuthLink(user discord.User) string {
	output, err := runGayauthCommand("generateDiscordAuthUrl", user)
	if err != nil {
		log.Fatalln(output, err)
	}
	return output
}

// isDiscordAuthenticated checks whether a user is authenticated
// in the database for Discord for a specific student type, and returns
// true if they are, or false otherwise.
func isDiscordAuthenticated(user discord.User, studentType StudentType) bool {
	output, err := runGayauthCommand("verifyDiscordAuth", user)
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() == 1 {
			log.Println(exitError)
			return false
		}
		log.Fatalln(output, err)
	}
	for _, code := range studentType.codes() {
		if code == output {
			return true
		}
	}
	return false
}

// runGayauthCommand runs a command with the artisan console.
func runGayauthCommand(command string, user discord.User) (string, error) {
	generatorCommand := exec.Command("./artisan", fmt.Sprintf("gayauth:%s", command), user.ID.String())

	var out bytes.Buffer
	generatorCommand.Stdout = &out

	err := generatorCommand.Run()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out.String()), nil
}
