# Nusa Agent Bot

Nusa Agent Bot is a dedicated Discord service built for the Nusa community. It provides automated member license verification, role synchronization with the central administration platform, and real-time market intelligence for the Indonesia Stock Exchange (IDX).

## Objectives

1. Automate Member Onboarding and Licensing: Streamline access control to private Discord channels by verifying activation tokens against the backend system, assigning appropriate roles (e.g., VIP, Member, Free), and revoking access upon subscription expiration.
2. Market Intelligence and Price Action Monitoring: Track Indonesia Stock Exchange (IDX) equities in real time, detecting price spikes, Auto Rejection Atas (ARA), Auto Rejection Bawah (ARB), and distributing structured summaries to target community channels.
3. Seamless Operational Integration: Maintain bi-directional synchronization between Discord and the central administration portal via background heartbeat workers, polling mechanisms, and event listeners.

---

## Core Capabilities

### 1. Member License Verification and Role Management
- Interactive Verification Modal: Members can click an interactive button to open a secure Discord modal for token entry.
- Administration Integration: Validates user-submitted tokens against the central administration platform, recording activation metadata (tier, duration, expiration).
- Automatic Role Assignment: Dynamically grants role privileges based on subscription tier (such as VIP) upon successful validation.
- Revocation Engine:
  - Remote Synchronization: Accepts remote revocation signals from the administration platform to instantly downgrade member privileges.
  - Background Poller: Periodically checks member license validity with the administration service, demoting users to the Free tier when subscriptions expire.

### 2. IDX Market Scanner
- Real-Time Market Feed: Fetches live equity prices, volume, and intraday statistics for Indonesia Stock Exchange tickers using Yahoo Finance feeds.
- Auto-Rejection Limit Calculation: Automatically calculates IDX auto-rejection thresholds based on regulatory price brackets:
  - Stock price below Rp 200: 35% limit
  - Stock price between Rp 200 and Rp 5,000: 25% limit
  - Stock price above Rp 5,000: 20% limit
- Gainers and Losers Broadcasts:
  - Top ARA / Gainers: Collects and ranks leading market gainers, identifying securities hitting upper rejection limits and publishing structured summaries to the designated channel.
  - Top ARB / Losers: Collects and ranks leading market decliners, flagging securities hitting lower limits and publishing reports to the designated channel.
- On-Demand Inquiries: Allows community members to look up any IDX ticker to view current price, net change, day high/low, open price, and traded volume in shares and lots.

### 3. Background Services and Resiliency
- Heartbeat and Auto-Reconnect: Continuously broadcasts agent status (`Online`, `Heartbeat`, `Offline`) to the administration service. Automatically handles temporary connection loss without interrupting Discord bot operations.
- Scheduled Market Scanner: Runs periodic market scans at configurable intervals during operational cycles.
- Graceful Shutdown: Captures termination signals (SIGINT, SIGTERM), informs the upstream backend of offline status, and safely closes active Discord and database connections.

---

## System Architecture

```text
+-------------------------+         +----------------------------+
|      Discord Guild      |         |     Central Admin Panel    |
| (Members, Channels, UI) |         |      Management Server     |
+------------+------------+         +--------------+-------------+
             |                                     |
             | WebSocket / Gateway                 | Synchronized State
             v                                     v
+----------------------------------------------------------------+
|                        Nusa Agent Bot                          |
|                                                                |
|  [ Discord Bot Layer ]                                         |
|    - Modal / Button interactions (Token submission)            |
|    - Slash Command router (/saham, /scan, /verif)               |
|    - Rich embed builders & interactive components              |
|                                                                |
|  [ Services & Workers ]                                        |
|    - Market Scanner (Yahoo Finance feed, ARA/ARB rules)        |
|    - Status Poller (User subscription validation)              |
|    - Heartbeat Worker (Auto-reconnecting upstream agent sync)  |
|                                                                |
|  [ Internal Service Layer ]                                    |
|    - Remote callback receiver & health monitoring              |
|                                                                |
|  [ Persistence Layer ]                                         |
|    - SQLite via GORM (Local user status cache)                 |
+----------------------------------------------------------------+
```

---

## Tech Stack

- Language: Go (1.22+)
- Discord Library: DiscordGo (`github.com/bwmarrin/discordgo`)
- Web Framework: Gin (`github.com/gin-gonic/gin`)
- CLI Framework: Cobra (`github.com/spf13/cobra`)
- ORM & Database: GORM with SQLite driver (`gorm.io/driver/sqlite`)
- External Data Providers: Yahoo Finance API (screener and chart endpoints)

---

## Project Structure

```text
agent-bot/
├── cmd/
│   ├── root.go              # Root Cobra command configuration
│   ├── install.go           # Database migration command
│   ├── scan.go              # Standalone scanner commands (scan, scan-ara, scan-arb)
│   └── start.go             # Main daemon runner (bot, services, workers)
├── internal/
│   ├── api/
│   │   ├── server.go        # Internal server and remote event handlers
│   │   └── server_test.go   # Server test suite
│   ├── bot/
│   │   ├── connect.go       # Discord session initialization and intents
│   │   ├── handler.go       # Message and interaction event handlers
│   │   ├── poll.go          # License expiration background poller
│   │   ├── scanner.go       # Embed generation and channel broadcasting
│   │   └── template/
│   │       └── modal.go     # Discord UI component definitions
│   ├── database/
│   │   └── database.go      # SQLite connection and GORM setup
│   ├── model/
│   │   └── models.go        # User entity and database schema
│   └── service/
│       ├── braint.go        # Central admin communication and heartbeat client
│       ├── braint_test.go   # Integration tests for admin service
│       ├── market.go        # Stock quote fetching and ARA/ARB calculation
│       └── market_test.go   # Unit tests for market formulas
├── makefile                 # Build automation targets
├── go.mod                   # Dependency manifest
└── readme.md                # Project documentation
```

---

## Configuration

The bot automatically loads a `.env` file from the working directory if present, and falls back to system environment variables and built-in defaults. A template is provided in `.env.example`.

| Variable | Description | Default |
|---|---|---|
| `DISCORD_TOKEN` | Discord Bot Token (Required) | - |
| `DISCORD_GUILD_ID` | Target Discord Guild ID for instant command registration (Optional) | Global (if omitted) |
| `ROLE_VIP_ID` | Discord Role ID for VIP subscription plan tier | `1546438682276528168` |
| `DEFAULT_MEMBER_ROLE_ID` | Default Discord Role ID assigned upon successful verification | `1546692502785101874` |
| `FREE_ROLE_ID` | Fallback Discord Role ID assigned when subscription expires | `1546692502785101874` |
| `TOP_ARA_CHANNEL_ID` | Discord channel ID for ARA / Top Gainers broadcasts | `1546695838779314226` |
| `TOP_ARB_CHANNEL_ID` | Discord channel ID for ARB / Top Losers broadcasts | `1546695999748050944` |
| `MOMENTUM_CHANNEL_ID` | Discord channel ID for Emiten Momentum broadcasts | `1546695838779314226` |
| `LARAVEL_API_URL` | Upstream administration service base URL | `http://127.0.0.1:8000/api/v1/agent` |
| `API_PORT` | Local REST API port for webhooks | `:8080` |
| `DATABASE_PATH` | Local SQLite database file path | `nusa.db` |

---

## Installation and Setup

### 1. Prerequisites
- Go version 1.22 or newer installed
- GCC or appropriate C compiler (required for SQLite CGO driver)

### 2. Clone and Dependencies
```bash
git clone <repository-url>
cd agent-bot
go mod download
```

### 3. Initialize Database
Run the installation command to perform database migrations and initialize local SQLite tables:
```bash
go run main.go install
```

### 4. Build Binary
Using make:
```bash
make build
```
Or directly using Go:
```bash
go build -o agent-bot main.go
```

---

## Usage

### Running the Complete Bot Daemon
Starts the Discord client, background scanner, status poller, and heartbeat worker:
```bash
./agent-bot start
```

To run the bot in standalone mode without connecting to the upstream administration backend:
```bash
./agent-bot start --no-laravel
```

### Running Standalone Scans (CLI)
You can trigger one-off scanner jobs without running the full daemon. These commands connect to Discord, execute the scan, send reports to the appropriate channel, and exit.

- Scan Top Gainers (ARA):
  ```bash
  ./agent-bot scan-ara
  ```
- Scan Top Losers (ARB):
  ```bash
  ./agent-bot scan-arb
  ```
- Scan Emiten Momentum (Volume Spike & 52W High Breakout):
  ```bash
  ./agent-bot scan-momentum
  ```
- Scan Both ARA, ARB, and Momentum:
  ```bash
  ./agent-bot scan
  ```

---

## Discord Commands Reference (Slash Commands)

| Command | Options | Description |
|---|---|---|
| `/saham` | `kode` (Required, Autocomplete) | Menampilkan live quote saham IDX dengan embed lengkap dan tombol interaktif **Refresh** (khusus pemanggil command). Contoh: `/saham kode:BBCA` atau `/saham kode:IHSG`. |
| `/scan ara` | - | Memindai Top 10 Gainers / ARA dan mempublikasikan laporannya ke channel ARA. Respon status command bersifat privat (ephemeral). |
| `/scan arb` | - | Memindai Top 10 Losers / ARB dan mempublikasikan laporannya ke channel ARB. Respon status command bersifat privat (ephemeral). |
| `/scan momentum` | - | Memindai radar **Emiten Momentum**: Unusual Volume Spike (>150% - 200% avg 10D) & 52-Week High Breakout ke channel Emiten Momentum. |
| `/scan all` | - | Memindai Top 10 ARA, ARB, dan Momentum sekaligus ke masing-masing channel. |
| `/verif` | - | Membuka modal input token Nusa untuk aktivasi role member secara langsung dan privat (ephemeral). |

---

## Testing

Execute the test suites across all packages:
```bash
go test -v ./...
```
