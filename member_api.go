package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/diamondburned/arikawa/v3/discord"
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
// true if they are, or false otherwise. It also returns the student
// type that the user does have in its second return.
func isDiscordAuthenticated(user discord.User, studentType StudentType) (bool, string) {
	output, err := runGayauthCommand("verifyDiscordAuth", user)
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() == 1 {
			return false, ""
		}
		log.Fatalln(output, err)
	}
	for _, code := range studentType.Codes() {
		if strings.EqualFold(code, output) {
			return true, output
		}
	}
	return false, output
}

// runGayauthCommand runs a command with the artisan console.
func runGayauthCommand(command string, user discord.User) (string, error) {
	artisan := filepath.Join(os.Getenv("AUTH_ROOT"), "artisan") // gets path to Laravel Artisan
	if _, err := os.Stat(artisan); err != nil {
		return "", fmt.Errorf("AUTH_ROOT is not set correctly or artisan is missing")
	}

	generatorCommand := exec.Command(artisan, fmt.Sprintf("gayauth:%s", command), user.ID.String())

	var out bytes.Buffer
	generatorCommand.Stdout = &out

	err := generatorCommand.Run()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out.String()), nil
}
