## Context

This is a greenfield Go CLI project for stock market monitoring and alerting. The tool will:
1. Fetch stock/index data from Alpha Vantage API
2. Evaluate user-defined alert conditions (e.g., "NASDAQ down 5% over 1 week")
3. Send Telegram notifications when conditions are met

**Constraints:**
- Alpha Vantage free tier: 25 API calls/day, 5 calls/minute
- Must work as a CLI tool (not a daemon/service initially)
- Configuration via .env and JSON/YAML config files

## Goals / Non-Goals

**Goals:**
- Simple CLI that can be run manually or via cron
- Support percentage-based alerts over configurable time periods (days, weeks)
- Support multiple stock symbols and indices
- Clean separation between data fetching, alert logic, and notifications
- Easy to extend with new notification channels

**Non-Goals:**
- Real-time streaming (would require premium API)
- Web UI or REST API
- Historical alert storage/analytics
- Support for other data providers (initially)

## Decisions

### 1. Project Structure
```
stock-market-notifier/
├── cmd/
│   └── notifier/
│       └── main.go           # CLI entry point
├── internal/
│   ├── alphavantage/
│   │   ├── client.go         # HTTP client for Alpha Vantage
│   │   └── types.go          # Response types
│   ├── alert/
│   │   ├── engine.go         # Alert evaluation logic
│   │   └── types.go          # Alert definition types
│   ├── telegram/
│   │   └── client.go         # Telegram Bot API client
│   └── config/
│       └── config.go         # Configuration loading
├── alerts.yaml               # Alert definitions
├── .env                      # API keys (gitignored)
├── .env.example              # Example env file
└── go.mod
```

**Rationale:** Standard Go project layout with `internal/` for encapsulation. Each capability maps to a package.

### 2. CLI Framework: Cobra
**Decision:** Use `github.com/spf13/cobra` for CLI

**Alternatives considered:**
- Standard library `flag`: Too basic for subcommands
- `urfave/cli`: Good but Cobra is more widely used in Go ecosystem

**Commands:**
- `notifier check` - Run alert checks once
- `notifier add-alert` - Add a new alert interactively
- `notifier list-alerts` - List configured alerts

### 3. Configuration: YAML + .env
**Decision:** 
- Use `.env` for secrets (API keys, bot tokens)
- Use `alerts.yaml` for alert definitions

**Alert definition format:**
```yaml
alerts:
  - symbol: "IXIC"        # NASDAQ Composite
    name: "NASDAQ Weekly Drop"
    condition:
      type: "percent_change"
      threshold: -5.0      # -5% = drop of 5%
      period: "7d"         # 7 days
    enabled: true
```

**Rationale:** YAML is human-readable for editing alerts. .env keeps secrets separate.

### 4. Alpha Vantage API Usage
**Decision:** Use `TIME_SERIES_DAILY` endpoint for daily close prices

**Rationale:** 
- Free tier supports this endpoint
- Daily granularity sufficient for multi-day alerts
- Can calculate percentage change from historical close prices

**API call pattern:**
```
GET https://www.alphavantage.co/query?function=TIME_SERIES_DAILY&symbol=IBM&apikey=XXX
```

### 5. Telegram Integration
**Decision:** Use direct HTTP calls to Telegram Bot API (no SDK)

**Rationale:** Simple use case (send message only) doesn't warrant a full SDK dependency.

**API call:**
```
POST https://api.telegram.org/bot<token>/sendMessage
{
  "chat_id": "<chat_id>",
  "text": "Alert: NASDAQ down 5.2% over the last 7 days",
  "parse_mode": "Markdown"
}
```

### 6. Error Handling
**Decision:** Fail gracefully per-alert, continue checking others

**Rationale:** One failed API call shouldn't stop all alerts from being checked.

## Risks / Trade-offs

| Risk | Mitigation |
|------|------------|
| Alpha Vantage rate limits (25/day free) | Cache responses, check limits in config, warn user |
| API key exposure | .env file with .gitignore, .env.example for documentation |
| Telegram bot setup complexity | Document bot creation steps in README |
| Index symbols may differ | Document correct symbols (e.g., IXIC for NASDAQ) |

## Open Questions

1. **Should we support a daemon mode?** - Defer to v2, cron is simpler for v1
2. **Multiple notification channels?** - Design allows it, implement Telegram first
3. **Alert cooldown period?** - Prevent duplicate alerts; consider for v1.1
