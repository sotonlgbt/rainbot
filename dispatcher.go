package main

import (
	"log"
	"reflect"
	"strings"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
)

// Dispatcher takes events in on its methods, and sends them to the Bot.
type Dispatcher struct {
	Bot Bot
}

// InteractionEventDispatcher fires when an InteractionCreateEvent occurs, and dispatches
// the relevant events to the bot.
func (d *Dispatcher) InteractionEventDispatcher(e *gateway.InteractionCreateEvent) {
	var err error
	switch data := e.Data.(type) {
	case *discord.CommandInteraction:
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

		switch data.Name {
		case "verification_button":
			err = d.Bot.CreateVerificationButton(e)
		case "pronoun_picker":
			err = d.Bot.CreatePronounPicker(e, *guild)
		case "colour_picker":
			err = d.Bot.CreateColourPicker(e, *guild)
		case "role_picker":
			err = d.Bot.CreateRolePicker(e, *guild)
		default:
			return
		}
	case *discord.ButtonInteraction:
		s := string(data.CustomID)
		switch {
		case strings.HasPrefix(s, colour_button_prefix):
			err = d.Bot.InteractionToggleUserRole(e, e.Member, strings.TrimPrefix(s, colour_button_prefix), e.GuildID, "requested colour role")
		case strings.HasPrefix(s, pronoun_button_prefix):
			err = d.Bot.InteractionToggleUserRole(e, e.Member, strings.TrimPrefix(s, pronoun_button_prefix), e.GuildID, "requested pronoun role")
		case strings.HasPrefix(s, role_button_prefix):
			err = d.Bot.InteractionToggleUserRole(e, e.Member, strings.TrimPrefix(s, role_button_prefix), e.GuildID, "requested generic role")
		case strings.HasPrefix(s, verify_button_guild_prefix):
			var guildSnowflake discord.Snowflake
			guildSnowflake, err = discord.ParseSnowflake(strings.TrimPrefix(s, verify_button_guild_prefix))
			if err != nil {
				log.Fatalln("Invalid guild ID found for button with text", s)
			}

			// Send an interaction as we start the verification process to acknowledge the button
			d.Bot.State.RespondInteraction(e.ID, e.Token, api.InteractionResponse{
				Type: api.MessageInteractionWithSource,
				Data: &api.InteractionResponseData{
					Content: option.NewNullableString("Verification started! Your verified role has been removed temporarily while you verify yourself - as soon as this is done, it'll be back 😄"),
					Flags:   api.EphemeralResponse,
				},
			})

			err = d.Bot.VerifyUser(*e.User, discord.GuildID(guildSnowflake))
		case s == "verifyme_button":
			err = d.Bot.OnVerifyMeButton(e)
		default:
			return
		}
	default:
		return
	}
	if err != nil {
		log.Println("Error in InteractionEventDispatcher for", reflect.TypeOf(e), "of error", err)
	}
}

// NewGuildMemberEventDispatcher fires when a new guild member joins.
func (d *Dispatcher) NewGuildMemberEventDispatcher(newMemberEvent *gateway.GuildMemberAddEvent) {
	err := d.Bot.VerifyUser(newMemberEvent.User, newMemberEvent.GuildID)
	if err != nil {
		log.Println("Error in NewGuildMemberEventDispatcher:", err)
	}
}
