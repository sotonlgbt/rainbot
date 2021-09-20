# Rainbot
The LGBTQ+soc bot for Discord.

## Setup
* Set `BOT_TOKEN` in the environment to the Discord token for the bot.
* Set `APP_ID` in the environment to the Discord app ID for the bot.
* Set `AUTH_ROOT` in the environment to the path to the root of the authentication system.
* Setup a cronjob to run Rainbot with `--reaperMode` however often channels should be checked for messages to delete based on the configuration in config.yml.

## Structure
**main.go** contains the application entry point, and sets up the handlers for the dispatcher system.
**dispatcher.go** handles events coming from Discord, and dispatches them to the other relevant parts of the code - usually the bot.
**bot.go** contains the core bot code - actually interacts with the user.
**member_api.go** handles verification of membership in conjunction with the LGBTQ+ Society authentication system.
**roles.go** contains the structures for the current student and alumni roles.
**config.go** contains the structures for the bot's configuration files.
**reaper.go** contains the code for the periodic message deletion system.