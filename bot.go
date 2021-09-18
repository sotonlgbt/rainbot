package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/bot"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
)

var verifiedRoles map[discord.GuildID]*discord.Role = map[discord.GuildID]*discord.Role{}

type Bot struct {
	// Context must not be embedded.
	Ctx *bot.Context
}

func (bot *Bot) Setup(sub *bot.Subcommand) {
	// Only allow people in guilds to run guildInfo.
	// sub.AddMiddleware("GuildInfo", middlewares.GuildOnly(bot.Ctx))
}

// // Help prints the default help message.
// func (bot *Bot) Help(*gateway.MessageCreateEvent) (string, error) {
// 	return bot.Ctx.Help(), nil
// }

// // Add demonstrates the usage of typed arguments. Run it with "~add 1 2".
// func (bot *Bot) Add(_ *gateway.MessageCreateEvent, a, b int) (string, error) {
// 	return fmt.Sprintf("%d + %d = %d", a, b, a+b), nil
// }

// Ping is a simple ping example, perhaps the most simple you could make it.
func (bot *Bot) Ping(*gateway.MessageCreateEvent) (string, error) {
	log.Println("Ponging")
	return "Pong!", nil
}

// // Say demonstrates how arguments.Flag could be used without the flag library.
// func (bot *Bot) Say(_ *gateway.MessageCreateEvent, f bot.RawArguments) (string, error) {
// 	if f != "" {
// 		return string(f), nil
// 	}
// 	return "", errors.New("missing content")
// }

func (bot *Bot) NewGuildMemberEventProcessor(newMemberEvent *gateway.GuildMemberAddEvent) error {
	memberChannel, err := bot.Ctx.CreatePrivateChannel(newMemberEvent.User.ID)
	if err != nil {
		return err
	}

	bot.Ctx.SendMessageComplex(memberChannel.ID, api.SendMessageData{
		Content: "Hi, welcome to the LGBTQ+ Society server! To verify that you're a student, please click here, and sign in within the next 10 minutes 😃",
		Components: []discord.Component{
			&discord.ActionRowComponent{
				Components: []discord.Component{
					&discord.ButtonComponent{
						Label: "Click here to verify",
						Emoji: &discord.ButtonEmoji{
							Name: "🔑",
						},
						Style: discord.LinkButton,
						URL:   getDiscordAuthLink(newMemberEvent.User),
					},
				},
			},
		},
	})

	bot.Ctx.SendMessageComplex(memberChannel.ID, api.SendMessageData{
		Content: "Once you've verified, please click here!",
		Components: []discord.Component{
			&discord.ActionRowComponent{
				Components: []discord.Component{
					&discord.ButtonComponent{
						Label:    "I've verified!",
						CustomID: "verified_button",
						Emoji: &discord.ButtonEmoji{
							Name: "✔️",
						},
						Style: discord.SuccessButton,
					},
				},
			},
		},
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*10)
	defer cancel()

	var interactionToRespondTo *gateway.InteractionCreateEvent

	// This might miss events that are sent immediately after. To make sure all
	// events are caught, ChanFor should be used.
	hasValidatedEvent := bot.Ctx.WaitFor(ctx, func(v interface{}) bool {
		// Incoming event is a component interaction:
		ci, ok := v.(*gateway.InteractionCreateEvent)
		if ok {
			if ci.Type == gateway.ButtonInteraction && ci.ChannelID == memberChannel.ID &&
				ci.User.ID == newMemberEvent.User.ID && ci.Data.CustomID == "verified_button" {
				interactionToRespondTo = ci
				return true
			}
		}

		// Incoming event is a message create event:
		mg, ok := v.(*gateway.MessageCreateEvent)
		if !ok {
			return false
		}

		// Message is from the same author and is a DM:
		return mg.Author.ID == newMemberEvent.User.ID && mg.ChannelID == memberChannel.ID
	})

	var memberType StudentType
	if newMemberEvent.GuildID.String() == alumni_server {
		memberType = &Alumnus{}
	} else {
		memberType = &CurrentStudent{}
	}

	if isDiscordAuthenticated(newMemberEvent.User, memberType) {
		verifiedRole, err := bot.getVerifiedRole(newMemberEvent.GuildID)
		if err != nil {
			return err
		}

		err = bot.Ctx.AddRole(newMemberEvent.GuildID, newMemberEvent.User.ID, *verifiedRole, api.AddRoleData{
			AuditLogReason: "Verified successfully with the bot",
		})
		if err != nil {
			return err
		}

		const message = "Thanks! You're now verified. Have a great day!"

		if interactionToRespondTo != nil {
			data := api.InteractionResponse{
				Type: api.UpdateMessage,
				Data: &api.InteractionResponseData{
					Content:    option.NewNullableString(message),
					Components: &[]discord.Component{},
				},
			}

			if err := bot.Ctx.RespondInteraction(interactionToRespondTo.ID, interactionToRespondTo.Token, data); err != nil {
				log.Println("failed to send interaction callback:", err)
			}
		} else {
			bot.Ctx.SendMessage(memberChannel.ID, message)
		}
	} else if hasValidatedEvent == nil { // timed out
		bot.Ctx.SendMessage(memberChannel.ID, "Whoops - time's up, and it doesn't look like you've verified. Please try joining the server again.")
		bot.Ctx.Kick(newMemberEvent.GuildID, newMemberEvent.User.ID, "Timed out without verification, took too long to verify")
		return errors.New("timed out waiting for response, kicked " + newMemberEvent.User.Username)
	} else {
		bot.Ctx.SendMessage(memberChannel.ID, "Sorry, that doesn't look like you authenticated successfully. Please try joining the server again.")
		bot.Ctx.Kick(newMemberEvent.GuildID, newMemberEvent.User.ID, "Was not authenticated but claimed to be")
		return errors.New("invalid claim of authentication, kicked " + newMemberEvent.User.Username)
	}

	return nil
}

// getVerifiedRole gets either the cached or the new "verified" role for the server.
func (bot *Bot) getVerifiedRole(guildID discord.GuildID) (*discord.RoleID, error) {
	if verifiedRoles[guildID] != nil {
		return &verifiedRoles[guildID].ID, nil
	}

	roles, err := bot.Ctx.Roles(guildID)
	if err != nil {
		return nil, err
	}

	for _, role := range roles {
		if strings.EqualFold(role.Name, "verified") {
			verifiedRoles[guildID] = &role
			return &role.ID, nil
		}
	}

	return nil, fmt.Errorf("no verified role found on server %d! Please ensure that there is a role on the server called 'verified', with case insensitive", guildID)
}

// // GuildInfo demonstrates the GuildOnly middleware done in (*Bot).Setup().
// func (bot *Bot) GuildInfo(m *gateway.MessageCreateEvent) (string, error) {
// 	g, err := bot.Ctx.GuildWithCount(m.GuildID)
// 	if err != nil {
// 		return "", fmt.Errorf("failed to get guild: %v", err)
// 	}

// 	return fmt.Sprintf(
// 		"Your guild is %s, and its maximum members is %d",
// 		g.Name, g.ApproximateMembers,
// 	), nil
// }

// // Repeat tells the bot to wait for the user's response, then repeat what they
// // said.
// func (bot *Bot) Repeat(m *gateway.MessageCreateEvent) (string, error) {
// 	_, err := bot.Ctx.SendMessage(m.ChannelID, "What do you want me to say?", nil)
// 	if err != nil {
// 		return "", err
// 	}

// 	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
// 	defer cancel()

// 	// This might miss events that are sent immediately after. To make sure all
// 	// events are caught, ChanFor should be used.
// 	v := bot.Ctx.WaitFor(ctx, func(v interface{}) bool {
// 		// Incoming event is a message create event:
// 		mg, ok := v.(*gateway.MessageCreateEvent)
// 		if !ok {
// 			return false
// 		}

// 		// Message is from the same author:
// 		return mg.Author.ID == m.Author.ID
// 	})

// 	if v == nil {
// 		return "", errors.New("timed out waiting for response")
// 	}

// 	ev := v.(*gateway.MessageCreateEvent)
// 	return ev.Content, nil
// }

// // Embed is a simple embed creator. Its purpose is to demonstrate the usage of
// // the ParseContent interface, as well as using the stdlib flag package.
// func (bot *Bot) Embed(_ *gateway.MessageCreateEvent, f arguments.Flag) (*discord.Embed, error) {
// 	fs := arguments.NewFlagSet()

// 	var (
// 		title  = fs.String("title", "", "Title")
// 		author = fs.String("author", "", "Author")
// 		footer = fs.String("footer", "", "Footer")
// 		color  = fs.String("color", "#FFFFFF", "Color in hex format #hhhhhh")
// 	)

// 	if err := f.With(fs.FlagSet); err != nil {
// 		return nil, err
// 	}

// 	if len(fs.Args()) < 1 {
// 		return nil, fmt.Errorf("usage: embed [flags] content...\n" + fs.Usage())
// 	}

// 	// Check if the color string is valid.
// 	if !strings.HasPrefix(*color, "#") || len(*color) != 7 {
// 		return nil, errors.New("invalid color, format must be #hhhhhh")
// 	}

// 	// Parse the color into decimal numbers.
// 	colorHex, err := strconv.ParseInt((*color)[1:], 16, 64)
// 	if err != nil {
// 		return nil, err
// 	}

// 	// Make a new embed
// 	embed := discord.Embed{
// 		Title:       *title,
// 		Description: strings.Join(fs.Args(), " "),
// 		Color:       discord.Color(colorHex),
// 	}

// 	if *author != "" {
// 		embed.Author = &discord.EmbedAuthor{
// 			Name: *author,
// 		}
// 	}
// 	if *footer != "" {
// 		embed.Footer = &discord.EmbedFooter{
// 			Text: *footer,
// 		}
// 	}

// 	return &embed, err
// }
