## Simple bot for Starting and Removing VMs hosted on Vultr with minecraft server inside

### What you need
- Vultr account and API key
- Telegram bot

### How to use
- Set chats allowed to use the bot in the `ALLOWED_CHATS` env
- Set the bot token in the `BOT_TOKEN` env
- Set the Vultr API key in the `VULTR_API_KEY` env
- Change rest of env variables to your needs 

### Commands
- `/create` - Start the VM and the minecraft server
- `/remove` - Stop the VM and destroy it