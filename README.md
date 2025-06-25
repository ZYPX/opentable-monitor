# OpenTable Monitor 🕵️‍♂️🍽️

A Go-based CLI and webhook tool that monitors OpenTable reservation availability and notifies you when a reservation opens up — perfect for high-demand restaurants.

## Features

- Real-time monitoring of OpenTable reservations
- CLI output with detailed time slot and seating type information
- Discord webhook integration for instant alerts
- Displays alternative time slots when the preferred one is unavailable
- Polls for updates every **1 minute**

## 🛠 Setup Instructions

### 1. Clone the Repository

```bash
git clone https://github.com/your-username/opentable-monitor.git
cd opentable-monitor
````

### 2. Install Go Dependencies

Ensure Go is installed. Then run:

```bash
go mod download
```

### 3. Configure Environment Variables

Copy the example `.env` file and replace placeholders:

```bash
cp .env.example .env
```

Edit `.env` and **replace the placeholder Discord webhook URL** with your own:

```
DISCORD_WEBHOOK_URL=https://discord.com/api/webhooks/your-webhook-id
```

## ▶️ Run the Monitor

Run the application with:

```bash
go run main.go
```

Follow the interactive prompts to select a restaurant and begin monitoring.
