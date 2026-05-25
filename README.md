# Limbo 🎬

Limbo is a lightweight, self-hosted Go dashboard and background notifier that tracks approved but unfulfilled requests from **Seerr** (Overseerr/Jellyseerr). It features a beautiful, responsive, glassmorphic Single-Page Application (SPA) web UI to browse, triage, and trigger search commands, while handling release date evaluation and Discord notifications.

---

## ✨ Features

- **Automated Sync & State Machine**: Periodically checks Seerr for approved requests, transitioning entries through status states:
  - `PENDING`: Released but not yet available or in the download client.
  - `WAITING_RELEASE`: Release date is in the future.
  - `NOT_AVAILABLE`: Manually marked as unavailable or unfulfillable.
  - `COMPLETED`: Automatically resolved when Seerr reports the media as partially or fully available.
- **Smart Release Evaluation**:
  - **Movies**: Evaluates release dates using priority: `Digital > Physical > Theatrical` based on country codes.
  - **TV Shows**: Parses show status and tracks individual season air dates.
- **Interactive Triage & Management**: Add notes, change states manually, or immediately trigger search commands in **Radarr** and **Sonarr**.
- **Discord Notifications**: Sends rich webhook embeds within a configurable release window.
- **Premium User Experience**: Designed using a modern glassmorphic dark theme, featuring subtle micro-animations, poster image loading, status badges, and full mobile responsiveness (PWA-ready).
- **Ultra-lightweight**: Written in Go with standard routing (`chi`), `gorm` (SQLite/PostgreSQL support), and embedded frontend assets—resulting in a tiny, performant Docker container (~15MB).

---

## 📂 Tech Stack

- **Backend**: Go 1.26.3, Chi Router, GORM
- **Database**: SQLite (default, WAL mode optimized) or PostgreSQL
- **Frontend**: Vanilla JS (SPA), Vanilla CSS (Glassmorphism & animations)
- **Asset Delivery**: HTML, JS, CSS embedded directly into the Go executable via `//go:embed`
- **Containerization**: Multi-stage CGO Alpine Dockerfile

---

## ⚙️ Configuration

All settings are configured using environment variables:

| Environment Variable | Description | Default | Required |
|----------------------|-------------|---------|----------|
| `DB_DRIVER` | Database engine (`sqlite` or `postgres`) | `sqlite` | No |
| `DB_DSN` | Connection DSN (SQLite file path or Postgres URI) | `limbo.db` | No |
| `SEERR_URL` | Internal Seerr API address | `http://localhost:5055` | No |
| `SEERR_PUBLIC_URL` | User-facing Seerr external address | `http://localhost:5055` | No |
| `SEERR_API_KEY` | Authenticated Seerr API key | None | **Yes** |
| `DISCORD_WEBHOOK_URL`| Destination webhook for release embeds | None | No |
| `RELEASE_COUNTRY` | 2-letter ISO country code for release priority | `US` | No |
| `SCAN_INTERVAL_MINUTES` | Frequency of background Seerr sync scans | `10` | No |
| `ALERT_THRESHOLD_MINUTES`| Age (in minutes) a request must be before triggering notifications | `10` | No |
| `ALERT_WINDOW_MINUTES` | Age window (in minutes) qualifying for notifications | `10` | No |
| `LIMBO_PORT` | Port for the HTTP web server | `3000` | No |
| `LOG_LEVEL` | Log verbosity (`debug`, `info`, `warn`, `error`) | `info` | No |

---

## 🚀 Quick Start

### 1. Docker Compose (Recommended)

Add Limbo to your `docker-compose.yaml` stack:

```yaml
services:
  limbo:
    image: limbo:latest
    container_name: limbo
    build:
      context: ./app
    restart: unless-stopped
    volumes:
      - ./config:/data
    ports:
      - "3800:3000"
    env_file:
      - .env
```

Create a `.env` file referencing the variables above:

```env
DB_DRIVER=sqlite
DB_DSN=/data/limbo.db
SEERR_URL=http://seerr:5055
SEERR_PUBLIC_URL=https://seerr.example.com
SEERR_API_KEY=your_seerr_api_key_here
DISCORD_WEBHOOK_URL=https://discord.com/api/webhooks/...
```

Run the container:
```bash
docker compose up -d --build
```

### 2. Local Development

Ensure Go 1.26+ is installed:


```bash
# Clone and enter the app directory
cd app

# Copy environment example
cp .env.example .env

# Run the app
go run main.go
```

The server will start on `http://localhost:3000`.

---

## 🧪 Testing

To run the automated test suite within a clean Docker environment:

```bash
docker build --target tester .
```

---

## 📄 License

This project is licensed under the GNU Affero General Public License v3 - see the `LICENSE` file for details.

