package main

import (
	"log"
	"strings"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
)

type Dispatcher struct {
	Bot Bot
}

// InteractionEventDispatcher fires when an InteractionCreateEvent occurs, and dispatches
// the relevant events to the bot.
func (d *Dispatcher) InteractionEventDispatcher(e *gateway.InteractionCreateEvent) {
	var err error
	if e.Type == gateway.CommandInteraction {
		if e.GuildID == 0 {
			// not in a guild? waa
			return
		}

		var guild *discord.Guild
		guild, err = d.Bot.State.Guild(e.GuildID)
		if err != nil {
			return
		}

		if guild.OwnerID != e.Member.User.ID {
			data := api.InteractionResponse{
				Type: api.MessageInteractionWithSource,
				Data: &api.InteractionResponseData{
					Content: option.NewNullableString("You're not authorised to run that command :c sorry! Ask your server owner."),
					Flags:   api.EphemeralResponse,
				},
			}

			if err = d.Bot.State.RespondInteraction(e.ID, e.Token, data); err != nil {
				log.Println("failed to send interaction callback for failed interaction:", err)
			}
			return
		}

		switch e.Data.Name {
		case "verification_button":
			err = d.Bot.CreateVerificationButton(e)
		case "pronoun_picker":
			err = d.Bot.CreatePronounPicker(e, *guild)
		case "colour_picker":
			err = d.Bot.CreateColourPicker(e, *guild)
		default:
			return
		}
	} else if e.Type == gateway.ButtonInteraction {
		if strings.HasPrefix(e.Data.CustomID, colour_button_prefix) {
			err = d.Bot.InteractionToggleUserRole(e, e.Member, strings.TrimPrefix(e.Data.CustomID, colour_button_prefix), e.GuildID, "requested colour role")
		} else if strings.HasPrefix(e.Data.CustomID, pronoun_button_prefix) {
			err = d.Bot.InteractionToggleUserRole(e, e.Member, strings.TrimPrefix(e.Data.CustomID, pronoun_button_prefix), e.GuildID, "requested pronoun role")
		} else {
			switch e.Data.CustomID {
			case "verifyme_button":
				err = d.Bot.OnVerifyMeButton(e)
			default:
				return
			}
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