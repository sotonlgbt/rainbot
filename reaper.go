package main

import (
	"fmt"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
)

// ReapChannelMessages takes a Discord channel ID, the amount of time to reap messages from, and the
// Discord API state. It then deletes messages older than the given duration in the given channel,
// excluding only messages that are pinned.
func (b *Bot) ReapChannelMessages(channel discord.ChannelID, reapDuration time.Duration) error {
	limit := time.Now().UTC().Add(-reapDuration)

	channelMessages, err := b.State.Messages(channel, 0)
	if err != nil {
		return err
	}

	reason := api.AuditLogReason(fmt.Sprintf("Reaping messages in channel %s before %s", channel, limit.Format(time.RFC822)))

	batchDeletionQueue := []discord.MessageID{}

	for i := len(channelMessages) - 1; i >= 0; i-- {
		if channelMessages[i].Timestamp.Time().UTC().Before(limit) {
			if !channelMessages[i].Pinned {
				// if it's been less than 13 days since the message was sent, we can queue it for batch deletion
				// technically, the limit is 14 days - but to avoid issues where we might be just on the cusp of
				// 14, this uses 13 for safety
				if time.Now().AddDate(0, 0, -13).Before(channelMessages[i].Timestamp.Time()) {
					batchDeletionQueue = append(batchDeletionQueue, channelMessages[i].ID)
				} else {
					// more than 13 days? delete the message one by one... :c
					err = b.State.DeleteMessage(channel, channelMessages[i].ID, reason)
					if err != nil {
						return err
					}
				}
			}
		} else {
			// it's now a waste of time processing any further messages, as we are in present territory!
			// yay :3
			break
		}
	}

	// these messages we can batch request - the 100 per request is handled by Arikawa
	err = b.State.DeleteMessages(channel, batchDeletionQueue, reason)
	return err
}
