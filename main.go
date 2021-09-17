package main

import (
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/diamondburned/arikawa/bot"
)

// To run, do `BOT_TOKEN="TOKEN HERE" go run .`

// alumni_server stores the ID of the alumni server.
var alumni_server string

func main() {
	var token = os.Getenv("BOT_TOKEN")
	if token == "" {
		log.Fatalln("No $BOT_TOKEN given.")
	}

	content, err := ioutil.ReadFile("alumni_server") // the file is inside the local directory
	if err == nil {
		alumni_server = strings.TrimSpace(string(content))
	}

	commands := &Bot{}

	wait, err := bot.Start(token, commands, func(ctx *bot.Context) error {
		ctx.HasPrefix = bot.NewPrefix("!", "~")
		ctx.EditableCommands = true

		// // Subcommand demo, but this can be in another package.
		// ctx.MustRegisterSubcommand(&Debug{})

		return nil
	})

	if err != nil {
		log.Fatalln(err)
	}

	log.Println("Bot started")

	// As of this commit, wait() will block until SIGINT or fatal. The past
	// versions close on call, but this one will block.
	// If for some reason you want the Cancel() function, manually make a new
	// context.
	if err := wait(); err != nil {
		log.Fatalln("Gateway fatal error:", err)
	}
}
