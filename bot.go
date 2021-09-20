package main

import (
	"fmt"
	"log"
	"math"
	"strings"
	"time"
	"unicode"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
)

var verifiedRoles map[discord.GuildID]*discord.Role = map[discord.GuildID]*discord.Role{}

// colour_button_prefix defines a prefix for the IDs on the buttons that set a person's colour roles.
const colour_button_prefix = "colour_button_"

// pronoun_button_prefix defines a prefix for the IDs on the buttons that set a person's pronoun roles.
const pronoun_button_prefix = "pronoun_button_"

// role_button_prefix defines a prefix for the IDs on the buttons that set a person's generic roles.
const role_button_prefix = "role_button_"

// Bot holds the current Discord state, and allows access to all of the bot's methods.
type Bot struct {
	State *state.State
}

// CreatePronounPicker is run by the interaction event dispatcher when the command
// to create a pronoun picker in the current channel is activated.
func (bot *Bot) CreatePronounPicker(e *gateway.InteractionCreateEvent, guild discord.Guild) error {
	// the event dispatcher has already checked we're in a guild, etc.

	if err := bot.State.RespondInteraction(e.ID, e.Token,
		generateInteractionResponseWithButtons(pronoun_button_prefix, config.Pronouns, "👋 What pronouns do you use?")); err != nil {
		log.Println("failed to send interaction callback in pronoun picker:", err)
		return err
	} else {
		return nil
	}
}

// CreateColourPicker is run by the interaction event dispatcher when the command
// to create a colour picker in the current channel is activated.
func (bot *Bot) CreateColourPicker(e *gateway.InteractionCreateEvent, guild discord.Guild) error {
	// the event dispatcher has already checked we're in a guild, etc.

	if err := bot.State.RespondInteraction(e.ID, e.Token,
		generateInteractionResponseWithButtons(colour_button_prefix, config.Guilds[guild.ID].Colours, "🎨 Pick a colour for your username!")); err != nil {
		log.Println("failed to send interaction callback in colour picker:", err)
		return err
	} else {
		return nil
	}
}

// CreateRolePicker is run by the interaction event dispatcher when the command
// to create a generic role picker in the current channel is activated.
func (bot *Bot) CreateRolePicker(e *gateway.InteractionCreateEvent, guild discord.Guild) error {
	// the event dispatcher has already checked we're in a guild, etc.

	if err := bot.State.RespondInteraction(e.ID, e.Token,
		generateInteractionResponseWithButtons(role_button_prefix, config.Guilds[guild.ID].Roles, "📋 Collect any extra roles you'd like.")); err != nil {
		log.Println("failed to send interaction callback in role picker:", err)
		return err
	} else {
		return nil
	}
}

// generateInteractionResponseWithButtons generates an InteractionResponse with a series of buttons,
// in the appropriate number of action rows.
func generateInteractionResponseWithButtons(prefix string, buttons []string, content string) api.InteractionResponse {
	actionRows := []discord.Component{}

	// Each row can only hold five components, so we need to do this for
	// ceil(n/5) times.
	for i := 0; i < int(math.Ceil(float64(len(buttons))/5)); i++ {
		actionRowComponents := []discord.Component{}

		limit := 5
		// Reduce the limit if we have less than 5 buttons left
		if len(buttons)-(i*5) < 5 {
			limit = len(buttons) - (i * 5)
		}

		// j will start at 0, then go up to 4. next time, it starts at 5
		for j := i * 5; j < (i*5)+limit; j++ {
			thisButton := buttons[j]

			// Golang has no built-in ability to just capitalise the first letter of a string...
			// :wut:
			// So we have to do it manually *sighs*
			thisButtonRuneArray := []rune(thisButton)
			thisButtonRuneArray[0] = unicode.ToUpper(thisButtonRuneArray[0])

			actionRowComponents = append(actionRowComponents, &discord.ButtonComponent{
				CustomID: prefix + thisButton,
				Label:    string(thisButtonRuneArray),
				Style:    discord.SecondaryButton,
			})
		}

		actionRows = append(actionRows, &discord.ActionRowComponent{
			Components: actionRowComponents,
		})
	}

	return api.InteractionResponse{
		Type: api.MessageInteractionWithSource,
		Data: &api.InteractionResponseData{
			Content:    option.NewNullableString(content),
			Components: &actionRows,
		},
	}
}

// CreateVerificationButton is run by the interaction event dispatcher when the command
// to create a verification button in the current channel is activated.
func (bot *Bot) CreateVerificationButton(e *gateway.InteractionCreateEvent) error {
	// the event dispatcher has already checked we're in a guild, etc.
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

	if err := bot.State.RespondInteraction(e.ID, e.Token, data); err != nil {
		log.Println("failed to send interaction callback:", err)
		return err
	} else {
		return nil
	}
}

// OnVerifyMeButton is run by the interaction event dispatcher when the "Verify me"
// button is pressed.
func (bot *Bot) OnVerifyMeButton(e *gateway.InteractionCreateEvent) error {
	verifiedRole, err := bot.getVerifiedRole(e.GuildID)
	if err != nil {
		return err
	}

	authenticated, memberType := isDiscordAuthenticated(e.Member.User, getMemberTypeForGuild(e.GuildID))
	if authenticated {
		// Optionally add the role if they aren't already owning it - so ignore errors here!
		bot.State.AddRole(e.GuildID, e.Member.User.ID, *verifiedRole, api.AddRoleData{
			AuditLogReason: api.AuditLogReason(fmt.Sprintf("Pre-registered, button verified with the bot as %s", memberType)),
		})

		data := api.InteractionResponse{
			Type: api.MessageInteractionWithSource,
			Data: &api.InteractionResponseData{
				Content: option.NewNullableString("You're already registered, so all's good! Enjoy your day 😊"),
				Flags:   api.EphemeralResponse,
			},
		}

		if err := bot.State.RespondInteraction(e.ID, e.Token, data); err != nil {
			log.Println("failed to send interaction callback for already-registered interaction:", err)
			return err
		} else {
			return nil
		}
	} else {
		bot.State.RemoveRole(e.GuildID, e.Member.User.ID, *verifiedRole, "Pressed the button to verify themselves, not entitled to verification yet")
		data := api.InteractionResponse{
			Type: api.MessageInteractionWithSource,
			Data: &api.InteractionResponseData{
				Content: option.NewNullableString("Check your DMs to complete verification!"),
				Flags:   api.EphemeralResponse,
			},
		}

		if err := bot.State.RespondInteraction(e.ID, e.Token, data); err != nil {
			log.Println("failed to send interaction callback for failed interaction:", err)
			return err
		} else {
			return bot.VerifyUser(e.Member.User, e.GuildID)
		}
	}
}

// InteractionToggleUserRole responds to an InteractionCreateEvent from the dispatcher by
// assigning a user a role, wrapping toggleUserRole.
func (bot *Bot) InteractionToggleUserRole(e *gateway.InteractionCreateEvent, member *discord.Member, roleName string, guildID discord.GuildID, auditLogReason string) error {
	assigned, err := bot.toggleUserRole(member, roleName, guildID, auditLogReason)
	if err != nil {
		return err
	}

	var message string
	if assigned {
		message = "Now you've got the"
	} else {
		message = "No more"
	}

	data := api.InteractionResponse{
		Type: api.MessageInteractionWithSource,
		Data: &api.InteractionResponseData{
			Content: option.NewNullableString(fmt.Sprintf("Nice job! %s %s role 😊", message, roleName)),
			Flags:   api.EphemeralResponse,
		},
	}

	if err := bot.State.RespondInteraction(e.ID, e.Token, data); err != nil {
		log.Println("failed to send interaction callback for assigning user role interaction:", err)
		return err
	} else {
		return nil
	}
}

// toggleUserRole internally assigns a user a role, or creates a role
// and assigns it to the user if it did not already exist. It returns
// a boolean indicating whether it assigned (true) or removed (false)
// the role, and an error.
func (bot *Bot) toggleUserRole(member *discord.Member, roleName string, guildID discord.GuildID, auditLogReason string) (bool, error) {
	roles, err := bot.State.Roles(guildID)
	if err != nil {
		return false, err
	}

	var roleToUse *discord.Role
	for _, role := range roles {
		if strings.EqualFold(role.Name, roleName) {
			roleToUse = &role
			// Failing to break early from the loop here creates an issue where the pointer moves to the end of the list.
			break
		}
	}

	if roleToUse == nil {
		roleToUse, err = bot.State.CreateRole(guildID, api.CreateRoleData{
			Name: roleName,
		})
		if err != nil {
			return false, err
		}
	}

	hasRole := false
	for _, roleID := range member.RoleIDs {
		if roleID == roleToUse.ID {
			hasRole = true
			break
		}
	}

	if hasRole {
		err = bot.State.RemoveRole(guildID, member.User.ID, roleToUse.ID, api.AuditLogReason(auditLogReason))
		return false, err
	} else {
		err = bot.State.AddRole(guildID, member.User.ID, roleToUse.ID, api.AddRoleData{
			AuditLogReason: api.AuditLogReason(auditLogReason),
		})
		return true, err
	}
}

// VerifyUser starts the verification process with a user, and manages it through to the end.
func (bot *Bot) VerifyUser(user discord.User, guildID discord.GuildID) error {
	memberChannel, err := bot.State.CreatePrivateChannel(user.ID)
	if err != nil {
		return err
	}

	bot.State.SendMessageComplex(memberChannel.ID, api.SendMessageData{
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
					&discord.ButtonComponent{
						Label: "Read our Member Data Policy",
						Emoji: &discord.ButtonEmoji{
							Name: "🔒",
						},
						Style: discord.LinkButton,
						URL:   "https://www.sotonlgbt.org.uk/privacy",
					},
				},
			},
		},
	})

	bot.State.SendMessageComplex(memberChannel.ID, api.SendMessageData{
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

	hasValidatedEventChannel, cancelEventChannel := bot.State.ChanFor(func(v interface{}) bool {
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
					bot.State.SendMessage(memberChannel.ID, "You've got five minutes left to verify - if you're not verified by then, you'll need to rejoin the server 😢 Having trouble? Message a committee member or email lgbt@soton.ac.uk.")
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
		err = bot.State.AddRole(guildID, user.ID, *verifiedRole, api.AddRoleData{
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

			if err := bot.State.RespondInteraction(interactionToRespondTo.ID, interactionToRespondTo.Token, data); err != nil {
				log.Println("failed to send interaction callback:", err)
			}
		} else {
			bot.State.SendMessage(memberChannel.ID, message)
		}
	} else {
		guildChannels, err := bot.State.Channels(guildID)
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

		invite, err := bot.State.CreateInvite(inviteChannel.ID, api.CreateInviteData{
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
			member, err := bot.State.Member(guildID, user.ID)
			if err != nil {
				return err
			}

			for _, roleID := range member.RoleIDs {
				if roleID == *verifiedRole {
					// the member was verified manually - ignore them
					bot.State.SendMessage(memberChannel.ID, "Looks like you were verified manually! Clipping through the map 😉 see ya!")
					return nil
				}
			}

			// the member doesn't have the verified role - kick them.
			reinviteMessageData.Content = "Whoops - time's up, and it doesn't look like you've verified. Please try joining the server again."
			bot.State.SendMessageComplex(memberChannel.ID, reinviteMessageData)
			bot.State.Kick(guildID, user.ID, "Timed out without verification, took too long to verify")
			return nil
		} else {
			if memberCode == "" {
				reinviteMessageData.Content = "Sorry, that doesn't look like you authenticated successfully. Please try joining the server again."
				bot.State.Kick(guildID, user.ID, "Claimed to be authenticated but was not in fact registered")
			} else {
				reinviteMessageData.Content = "Hmm - that doesn't look like you have the right type of University account for this server. If you've recently graduated, you may need to get a committee member to manually verify you (lgbt@soton.ac.uk), or contact iSolutions to get them to correct your account. Otherwise, please try joining the server again."
				bot.State.Kick(guildID, user.ID, api.AuditLogReason(fmt.Sprintf("Was not authenticated successfully - authenticated as %s which is invalid for this guild", memberCode)))
			}
			bot.State.SendMessageComplex(memberChannel.ID, reinviteMessageData)
			return nil
		}
	}

	return nil
}

// getVerifiedRole gets either the cached or the new "verified" role for the server.
func (bot *Bot) getVerifiedRole(guildID discord.GuildID) (*discord.RoleID, error) {
	if verifiedRoles[guildID] != nil {
		return &verifiedRoles[guildID].ID, nil
	}

	roles, err := bot.State.Roles(guildID)
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
	memberType = &CurrentStudent{}

	for configGuildID, guildConfig := range config.Guilds {
		if guildID == configGuildID && guildConfig.AlumniGuild {
			memberType = &Alumnus{}
		}
	}

	return memberType
}
