package main

import (
	"log"

	"github.com/diamondburned/arikawa/v3/gateway"
)

type Dispatcher struct {
	Bot Bot
}

// InteractionEventDispatcher fires when an InteractionCreateEvent occurs, and dispatches
// the relevant events to the bot.
func (d *Dispatcher) InteractionEventDispatcher(e *gateway.InteractionCreateEvent) {
	var err error
	if e.Type == gateway.CommandInteraction {
		switch e.Data.Name {
		case "verification_button":
			err = d.Bot.CreateVerificationButton(e)
		default:
			return
		}
	} else if e.Type == gateway.ButtonInteraction {
		switch e.Data.CustomID {
		case "verifyme_button":
			err = d.Bot.OnVerifyMeButton(e)
		default:
			return
		}
	} else {
		return
	}
	if err != nil {
		log.Println("Error in InteractionEventDispatcher for", e.Type, "of error", err)
	}
}

// NewGuildMemberEventDispatcher fires when a new guild member joins.
func (d *Dispatcher) NewGuildMemberEventDispatcher(newMemberEvent *gateway.GuildMemberAddEvent) {
	err := d.Bot.VerifyUser(newMemberEvent.User, newMemberEvent.GuildID)
	if err != nil {
		log.Println("Error in NewGuildMemberEventDispatcher:", err)
	}
}
