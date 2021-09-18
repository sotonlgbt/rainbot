package main

import (
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
	return bot.verifyUser(newMemberEvent.User, newMemberEvent.GuildID)
}

func (bot *Bot) InteractionCreateEventProcessor(e *gateway.InteractionCreateEvent) error {
	if e.Type == gateway.CommandInteraction && e.Data.Name == "verification_button" {
		if e.GuildID == 0 {
			// not in a guild? waa
			return nil
		}
		guild, err := bot.Ctx.Guild(e.GuildID)
		if err != nil {
			return nil
		}
		if guild.OwnerID != e.Member.User.ID {
			data := api.InteractionResponse{
				Type: api.MessageInteractionWithSource,
				Data: &api.InteractionResponseData{
					Content: option.NewNullableString("You're not authorised to run that command :c sorry! Ask your server owner."),
					Flags:   api.EphemeralResponse,
				},
			}

			if err := bot.Ctx.RespondInteraction(e.ID, e.Token, data); err != nil {
				log.Println("failed to send interaction callback for failed interaction:", err)
				return err
			} else {
				return nil
			}
		} else {
			log.Println("doing the thing")

			data := api.InteractionResponse{
				Type: api.MessageInteractionWithSource,
				Data: &api.InteractionResponseData{
					Content: option.NewNullableString("Ready to get verified? Click here to start the process..."),
					Components: &[]discord.Component{
						&discord.ActionRowComponent{
							Components: []discord.Component{
								&discord.ButtonComponent{
									CustomID: "verifyme_button",
									Label:    "Let's get verified!",
									Emoji: &discord.ButtonEmoji{
										Name: "🎉",
									},
									Style: discord.PrimaryButton,
								},
							},
						},
					},
				},
			}

			if err := bot.Ctx.RespondInteraction(e.ID, e.Token, data); err != nil {
				log.Println("failed to send interaction callback:", err)
				return err
			} else {
				return nil
			}
		}
	} else if e.Type == gateway.ButtonInteraction && e.Data.CustomID == "verifyme_button" {
		verifiedRole, err := bot.getVerifiedRole(e.GuildID)
		if err != nil {
			return err
		}

		authenticated, _ := isDiscordAuthenticated(e.Member.User, getMemberTypeForGuild(e.GuildID))
		if authenticated {
			data := api.InteractionResponse{
				Type: api.MessageInteractionWithSource,
				Data: &api.InteractionResponseData{
					Content: option.NewNullableString("You're already registered, so all's good! Enjoy your day 😊"),
					Flags:   api.EphemeralResponse,
				},
			}

			if err := bot.Ctx.RespondInteraction(e.ID, e.Token, data); err != nil {
				log.Println("failed to send interaction callback for already-registered interaction:", err)
				return err
			} else {
				return nil
			}
		} else {
			bot.Ctx.RemoveRole(e.GuildID, e.Member.User.ID, *verifiedRole, "Pressed the button to verify themselves, not entitled to verification yet")
			data := api.InteractionResponse{
				Type: api.MessageInteractionWithSource,
				Data: &api.InteractionResponseData{
					Content: option.NewNullableString("Check your DMs to complete verification!"),
					Flags:   api.EphemeralResponse,
				},
			}

			if err := bot.Ctx.RespondInteraction(e.ID, e.Token, data); err != nil {
				log.Println("failed to send interaction callback for failed interaction:", err)
				return err
			} else {
				return bot.verifyUser(e.Member.User, e.GuildID)
			}
		}
	} else {
		return nil
	}
}

func (bot *Bot) verifyUser(user discord.User, guildID discord.GuildID) error {
	memberChannel, err := bot.Ctx.CreatePrivateChannel(user.ID)
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
						URL:   getDiscordAuthLink(user),
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

	var interactionToRespondTo *gateway.InteractionCreateEvent

	hasValidatedEventChannel, cancelEventChannel := bot.Ctx.ChanFor(func(v interface{}) bool {
		// Incoming event is a component interaction:
		ci, ok := v.(*gateway.InteractionCreateEvent)
		if ok {
			if ci.Type == gateway.ButtonInteraction && ci.ChannelID == memberChannel.ID &&
				ci.User.ID == user.ID && ci.Data.CustomID == "verified_button" {
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
		return mg.Author.ID == user.ID && mg.ChannelID == memberChannel.ID
	})

	halfElapsed := false
	timedOut := false
repeatSelect:
	for {
		select {
		case <-hasValidatedEventChannel:
			break repeatSelect
		case <-time.After(time.Minute * 5):
			if halfElapsed {
				timedOut = true
				break repeatSelect
			} else {
				isAuthenticated, _ := isDiscordAuthenticated(user, getMemberTypeForGuild(guildID))
				if isAuthenticated {
					break repeatSelect
				} else {
					halfElapsed = true
					bot.Ctx.SendMessage(memberChannel.ID, "You've got five minutes left to verify - if you're not verified by then, you'll need to rejoin the server 😢 Having trouble? Message a committee member or email lgbt@soton.ac.uk.")
				}
			}
		}
	}

	cancelEventChannel()

	verifiedRole, err := bot.getVerifiedRole(guildID)
	if err != nil {
		return err
	}

	isAuthenticated, memberCode := isDiscordAuthenticated(user, getMemberTypeForGuild(guildID))
	if isAuthenticated {
		err = bot.Ctx.AddRole(guildID, user.ID, *verifiedRole, api.AddRoleData{
			AuditLogReason: api.AuditLogReason(fmt.Sprintf("Verified successfully with the bot as %s", memberCode)),
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
	} else {
		guildChannels, err := bot.Ctx.Channels(guildID)
		if err != nil {
			return err
		}

		var inviteChannel discord.Channel
		// start off with a ridiculous position - our first channel must be below this.
		var lowestChannelPosition int = 10000
		for _, channel := range guildChannels {
			if channel.Type == discord.GuildText && lowestChannelPosition > channel.Position {
				inviteChannel = channel
				lowestChannelPosition = channel.Position
			}
		}

		invite, err := bot.Ctx.CreateInvite(inviteChannel.ID, api.CreateInviteData{
			MaxUses:        1,
			Unique:         true,
			AuditLogReason: api.AuditLogReason(fmt.Sprintf("%s failed authentication, so creating them an easy re-invite link", user.Username)),
		})
		if err != nil {
			return err
		}

		reinviteMessageData := api.SendMessageData{
			Components: []discord.Component{
				&discord.ActionRowComponent{
					Components: []discord.Component{
						&discord.ButtonComponent{
							Label: "Let's try again",
							Emoji: &discord.ButtonEmoji{
								Name: "😢",
							},
							Style: discord.LinkButton,
							URL:   fmt.Sprintf("https://discord.gg/%s", invite.Code),
						},
					},
				},
			},
		}

		if timedOut {
			member, err := bot.Ctx.Member(guildID, user.ID)
			if err != nil {
				return err
			}

			for _, roleID := range member.RoleIDs {
				if roleID == *verifiedRole {
					// the member was verified manually - ignore them
					bot.Ctx.SendMessage(memberChannel.ID, "Looks like you were verified manually! Clipping through the map 😉 see ya!")
					return nil
				}
			}

			// the member doesn't have the verified role - kick them.
			reinviteMessageData.Content = "Whoops - time's up, and it doesn't look like you've verified. Please try joining the server again."
			bot.Ctx.SendMessageComplex(memberChannel.ID, reinviteMessageData)
			bot.Ctx.Kick(guildID, user.ID, "Timed out without verification, took too long to verify")
			return errors.New("timed out waiting for response, kicked " + user.Username)
		} else {
			reinviteMessageData.Content = "Sorry, that doesn't look like you authenticated successfully. Please try joining the server again."
			bot.Ctx.SendMessageComplex(memberChannel.ID, reinviteMessageData)
			if memberCode == "" {
				bot.Ctx.Kick(guildID, user.ID, "Claimed to be authenticated but was not in fact registered")
			} else {
				bot.Ctx.Kick(guildID, user.ID, api.AuditLogReason(fmt.Sprintf("Was not authenticated successfully - authenticated as %s which is invalid for this guild", memberCode)))
			}
			return errors.New("invalid claim of authentication, kicked " + user.Username)
		}
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

// getMemberTypeForGuild takes a guild ID and gets the type of
// student meant to be on that guild.
func getMemberTypeForGuild(guildID discord.GuildID) StudentType {
	var memberType StudentType
	if guildID.String() == alumni_server {
		memberType = &Alumnus{}
	} else {
		memberType = &CurrentStudent{}
	}
	return memberType
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
