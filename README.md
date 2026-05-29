<p align="center">
  <img src="frontend/assets/logo.svg" alt="Limbo Logo" width="64" height="64">
</p>

<h1 align="center"><font size="7">Limbo</font></h1>

<p align="center">
  <strong>A lightweight, self-hosted Go dashboard and background notifier that tracks approved but unfulfilled requests from Seerr.</strong>
</p>

<p align="center">
  <a href="#-features">Features</a> &bull;
  <a href="#-getting-started">Getting Started</a> &bull;
  <a href="#-configuration">Configuration</a> &bull;
  <a href="#-development">Development</a>
</p>

---

Limbo is a lightweight, self-hosted Go dashboard and background notifier that tracks approved but unfulfilled requests from **Seerr** (Overseerr/Jellyseerr). It features a beautiful, responsive, glassmorphic Single-Page Application (SPA) web UI to browse and triage requests, while handling release date evaluation and Discord notifications.

<p align="center">
  <img src=".github/screenshots/dashboard-dark.png" alt="Limbo Dashboard Dark Mode" width="800">
</p>

<details>
  <summary>📸 View Light Mode & Maintenance Screens</summary>
  <br>
  <p align="center">
    <strong>Light Mode Dashboard</strong><br><br>
    <img src=".github/screenshots/dashboard-light.png" alt="Limbo Dashboard Light Mode" width="800">
  </p>
  <p align="center">
    <strong>Maintenance Operations</strong><br><br>
    <img src=".github/screenshots/maintenance.png" alt="Limbo Triage & Maintenance Details" width="800">
  </p>
</details>

---

## ✨ Features

- **Automated Sync & State Machine**: Periodically checks Seerr for approved requests, transitioning entries through statuses.
- **Smart Release Evaluation**:
  - **Movies**: Evaluates release dates using priority: `Digital > Physical > Theatrical` based on country codes.
  - **TV Shows**: Parses show status and tracks individual season air dates.
- **Manual State Overrides & Reasons**: Manage and override request statuses (e.g., parking a request as `UNAVAILABLE` or reverting to `PENDING`) and add administrator cancellation/unavailable reasons.
- **Maintenance & Cache Operations**: Purge stale requests older than a specific threshold (clearing them from both Limbo and Seerr), refresh the TMDB poster and release date cache, and trigger test Discord notifications.
- **Discord Notifications**: Sends rich webhook embeds within a configurable release window.
- **Premium User Experience**: Designed using a modern glassmorphic dark theme, featuring subtle micro-animations, poster image loading, status badges, and full mobile responsiveness (PWA-ready).
- **Ultra-lightweight**: Written in Go with standard routing (`chi`), `gorm` (SQLite/PostgreSQL support), and embedded frontend assets—resulting in a tiny, performant Docker container (~15MB).

---

## 📂 Tech Stack

- **Backend**: Go 1.26.3, Chi Router, GORM
- **Database**: SQLite (default, WAL mode optimized) or PostgreSQL
- **Frontend**: Vanilla JS (SPA), Vanilla CSS (Glassmorphism & animations)
- **Asset Delivery**: HTML, JS, CSS embedded directly into the Go executable via `//go:embed`
- **Containerization**: Multi-stage CGO Alpine Dockerfile (local development) and lightweight runtime release Dockerfile (packaging static host-compiled binaries for CI/CD)

---

## ⚙️ Configuration

All settings are configured using environment variables:

| Environment Variable | Description | Default | Required |
|----------------------|-------------|---------|----------|
| `DB_DRIVER` | Database engine (`sqlite` or `postgres`) | `sqlite` | No |
| `POSTGRES_URL` | PostgreSQL connection URL (e.g. `postgresql://user:pass@host:5432/db`) | None | Yes (only if `DB_DRIVER` is `postgres`) |
| `SQLITE_PATH` | SQLite database file path | `/data/limbo.db` | No |
| `SEERR_URL` | Internal Seerr API address | `http://localhost:5055` | No |
| `SEERR_PUBLIC_URL` | User-facing Seerr external address | `http://localhost:5055` | No |
| `SEERR_API_KEY` | Authenticated Seerr API key | None | **Yes** |
| `DISCORD_WEBHOOK_URL`| Destination webhook for release embeds | None | No |
| `RELEASE_COUNTRY` | 2-letter ISO country code for release priority | `US` | No |
| `SCAN_INTERVAL_MINUTES` | Frequency of background Seerr sync scans | `10` | No |
| `ALERT_DELAY_MINUTES` | Minimum request age (in minutes) before Discord notification (gives automation time to download first) | `10` | No |
| `ALERT_MAX_AGE_MINUTES` | Maximum request age (in minutes) to qualify for notifications (prevents spamming old requests) | `1440` | No |
| `LIMBO_PORT` | Port for the HTTP web server | `3000` | No |
| `LOG_LEVEL` | Log verbosity (`debug`, `info`, `warn`, `error`) | `info` | No |

### 🔒 Secrets & `_FILE` Environment Variables

For sensitive configuration fields, you can append `_FILE` to the variable name to load its value from a file (e.g. Docker Secrets or Kubernetes secrets).
Supported variables:
- `POSTGRES_URL_FILE` (defaults to `/run/secrets/postgres_url`)
- `SEERR_API_KEY_FILE` (defaults to `/run/secrets/seerr_api_key`)
- `DISCORD_WEBHOOK_URL_FILE` (defaults to `/run/secrets/discord_webhook_url`)

**Note**: Either the direct variable or the `_FILE` variable must be used. If both are set simultaneously, the application will fail validation on startup. If neither is set, the application automatically attempts to load the default `/run/secrets/...` file path.

---

## 🚀 Quick Start

### 1. Docker Compose (Recommended)

Add Limbo to your `docker-compose.yaml` stack:

```yaml
services:
  limbo:
    image: ghcr.io/smark91/limbo:latest
    container_name: limbo
    restart: unless-stopped
    volumes:
      - ./config:/data
    ports:
      - "${LIMBO_PORT:-3000}:${LIMBO_PORT:-3000}"
    env_file:
      - .env
    # BELOW CONFIG ONLY IF USING FILE SECRETS
    secrets:
      - seerr_api_key
      - postgres_url
      - discord_webhook_url

secrets:
  seerr_api_key:
    file: ./secrets/seerr_api_key
  postgres_url:
    file: ./secrets/postgres_url
  discord_webhook_url:
    file: ./secrets/discord_webhook_url
```

> [!WARNING]
> If you change the internal `LIMBO_PORT` (for example, to `8080`), the container's internal service port changes. In this case, you must also update the `HEALTHCHECK` port defined in the `Dockerfile` (or override it in `docker-compose.yaml`) to match the new port so the health check endpoint remains reachable:
> 
> ```yaml
>     healthcheck:
>       test: wget -qO- "http://127.0.0.1:${LIMBO_PORT:-3000}/api/health" | grep '"status":"ok"' || exit 1
>       interval: 30s
>       timeout: 10s
>       retries: 3
> ```

Copy the example environment file to create your own configuration:

```bash
cp .env.example .env
```

Edit `.env` as needed to set your values.

Run the container:
```bash
docker compose up -d
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

