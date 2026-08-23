# Stock Drop Notifier

**Get a Telegram message when a stock drops. Free, forever, and you don't even need a
machine of your own to run it.**

[![CI](https://github.com/ajinux/stock-drop-notifier/actions/workflows/ci.yml/badge.svg)](https://github.com/ajinux/stock-drop-notifier/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/badge/go-1.25%2B-00ADD8?logo=go)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

A single Go binary that watches the stocks you care about and pings your Telegram
when one of them moves past a threshold you set. No account, no subscription, no
server to rent. Fork it, add three secrets, and GitHub Actions runs the checks for
you on a schedule — or run it locally if you'd rather keep everything on your own box.

```
⚠️ *AAPL Weekly Drop* Alert

*AAPL Weekly Drop* (AAPL) is 📉 down *6.42%* from peak ($241.30) over the last 7 days

• Current: $225.81
• Peak Price: $241.30
• Threshold: -5.00%
• Time: 2026-08-04 18:00:03 IST
```

## Why

I work full time in tech and check the market maybe once or twice a month. Every drop
reaches me the same way: as old news. The stock fell 12% on a Tuesday, recovered most of
it by the following week, and by the time I opened a chart the chance to buy it cheap had
already passed.

So this tool watches instead of me, and it measures drops **from the period's peak**
rather than start-to-end — because a dip that recovers is exactly the event a
start-to-end comparison erases. That's the whole idea. Everything else is plumbing.

Most price-alert services want an account, an app, and eventually a subscription.
This one is a ~10 MB binary plus a YAML file. The two services it depends on —
[Alpha Vantage](https://www.alphavantage.co) for prices and
[Telegram Bots](https://core.telegram.org/bots) for delivery — are both free at
the volume a personal watchlist needs.

**What it does**

- Watches any symbol Alpha Vantage covers: US stocks, ETFs, international tickers
- Triggers on percentage drops **or** gains over a period you define (`7d`, `1m`, `1y`, …)
- Measures the drop **from the period's peak** by default — the way you'd actually
  describe a dip — or start-to-end if you prefer
- Sends a formatted Telegram message per triggered alert
- Runs on demand, or on a schedule via GitHub Actions, cron, or launchd
- Ships an AI-assistant skill ([`skills/stock-research.md`](skills/stock-research.md)) that
  researches stocks worth watching, if you don't already know what to add

**What it does not do (yet)**

- Intraday prices — it works off daily closes
- Anything other than percentage change (no volume, RSI, moving averages)
- Notification channels other than Telegram

## How it works

```
alerts.yaml ──▶ notifier check ──▶ Alpha Vantage (daily closes)
                      │
                      ├─ compute % change over the period
                      ├─ compare against your threshold
                      └─ triggered? ──▶ Telegram bot ──▶ your chat
```

Each `check` run is stateless: it reads your alerts, fetches the data, evaluates,
notifies, and exits. Nothing is stored between runs, so you can run it as often as
your API quota allows.

## Quick start

**Prerequisites:** Go 1.25+ and a Telegram account.

### 1. Install

```bash
go install github.com/ajinux/stock-drop-notifier/cmd/notifier@latest
```

Or build from source, which is what you want if you plan to change anything:

```bash
git clone https://github.com/ajinux/stock-drop-notifier.git
cd stock-drop-notifier
make build          # produces ./notifier
```

Either way, `notifier` reads its config from whatever directory you run it in — so
pick a directory to keep `.env` and `alerts.yaml` in, and run it from there.

### 2. Get an Alpha Vantage API key

Go to [alphavantage.co](https://www.alphavantage.co/support/#api-key) and grab an API key — it takes
about thirty seconds and no payment details. Not even email verification is required. The free tier gives you **25 requests per
day**, which covers 25 alerts checked once daily.

### 3. Create a Telegram bot

1. Message [@BotFather](https://t.me/BotFather) on Telegram, send `/newbot`, and follow
   the prompts. Save the token it gives you — it looks like
   `123456789:ABCdefGHIjklMNOpqrsTUVwxyz`.
2. Send any message to your new bot (a bot cannot start a conversation with you).
3. Get your numeric chat ID by messaging [@userinfobot](https://t.me/userinfobot), or by
   opening `https://api.telegram.org/bot<YOUR_TOKEN>/getUpdates` and reading `chat.id`
   from the JSON. Copy the digits only — a stray space or newline is rejected with
   `invalid TELEGRAM_CHAT_ID`.

### 4. Configure

```bash
cp .env.example .env
```

```bash
ALPHAVANTAGE_API_KEY=your_api_key_here
TELEGRAM_BOT_TOKEN=your_bot_token_here   # optional
TELEGRAM_CHAT_ID=your_chat_id_here       # optional, numeric
```

Only the API key is required. Without the Telegram values, `check` still evaluates every
alert and prints the triggered ones to stdout — useful for a first run or a dry test.

Verify delivery end to end:

```bash
./notifier test-telegram
```

### 5. Add an alert and run it

The repo ships a working `alerts.yaml` using free-tier-friendly ETFs — edit it directly,
or start over from `alerts.yaml.example`:

```bash
cp alerts.yaml.example alerts.yaml
```

Note that `alerts.yaml` is a tracked file, so your edits will show up in `git status`.
That's deliberate: the [GitHub Actions workflow](#github-actions-no-machine-required)
reads your watchlist from the repo.

Or build it up one alert at a time:

```bash
./notifier add --symbol AAPL --threshold -5 --period 7d --name "AAPL Weekly Drop"
./notifier check --verbose
```

```
Evaluating 1 alert(s)...

🔔  AAPL Weekly Drop (AAPL): TRIGGERED (Current: $225.81, Peak: $241.30, Change: -6.42%, Threshold: -5.00%)

Sent 1 notification(s)

Summary: 1 evaluated, 1 triggered, 0 errors
```

> **Note:** `notifier` reads `.env` and `alerts.yaml` from the **current working
> directory**. Run it from the project directory, or `cd` there first in your cron job.

## Commands

| Command | Description |
|---------|-------------|
| `notifier check` | Evaluate enabled alerts and notify on the triggered ones |
| `notifier list` | Show all configured alerts in a table |
| `notifier add` | Add an alert to `alerts.yaml` |
| `notifier remove <name>` | Remove an alert by name |
| `notifier test-telegram` | Send a mock alert to verify your Telegram setup |
| `notifier version` | Print version, commit, and build date |

### `check`

| Flag | Short | Description |
|------|-------|-------------|
| `--verbose` | `-v` | Print the evaluation result for every alert, not just triggered ones |
| `--limit <n>` | `-l` | Evaluate only the first *n* enabled alerts — handy for staying under the daily quota |

### `add`

| Flag | Short | Description |
|------|-------|-------------|
| `--symbol` | `-s` | Ticker, e.g. `AAPL` (required) |
| `--threshold` | `-t` | Percentage; negative for a drop, positive for a gain (required) |
| `--period` | `-p` | Lookback window, e.g. `7d` (required) |
| `--name` | `-n` | Unique alert name (required) |
| `--enabled` | `-e` | Whether the alert is active (default `true`) |
| `--method` | `-m` | `from_peak` (default) or `period_to_period` |

```bash
# Alert when AAPL falls 5% from its 7-day peak
./notifier add -s AAPL -t -5 -p 7d -n "AAPL Weekly Drop"

# Alert when MSFT is up 10% over 30 days, measured start to end
./notifier add -s MSFT -t 10 -p 30d -n "MSFT Monthly Gain" -m period_to_period

# Add it disabled, to enable later by editing alerts.yaml
./notifier add -s GOOGL -t -20 -p 1y -n "GOOGL Yearly Drop" -e=false
```

### `list`

```
NAME               SYMBOL  THRESHOLD  PERIOD  METHOD            ENABLED
----               ------  ---------  ------  ------            -------
AAPL Weekly Drop   AAPL    -5.0%      7d      from_peak         Yes
MSFT Monthly Gain  MSFT    10.0%      30d     period_to_period  Yes
GOOGL Yearly Drop  GOOGL   -20.0%     1y      from_peak         No
```

## Alert configuration

Alerts live in `alerts.yaml`. The CLI writes this file, but editing it by hand is
equally fine — see `alerts.yaml.example` for a starting point.

```yaml
alerts:
  - name: "AAPL Weekly Drop"
    symbol: "AAPL"
    condition:
      type: "percent_change"
      threshold: -5.0                      # negative = drop, positive = gain
      period: "7d"
      calculation_method: "from_peak"      # optional, this is the default
    enabled: true
```

### Threshold

A **negative** threshold fires when the price falls by at least that much; a
**positive** threshold fires when it rises by at least that much. `-5.0` means
"tell me when it's down 5% or more."

### Calculation method

This is the setting worth understanding, because it changes what "down 5%" means.

**`from_peak` (default)** — finds the highest close within the period and measures the
current price against it. A stock that ran up to $100 and slid back to $94 is *down 6%*.
This is the intuitive reading of "it dropped," and it catches dips that a start-to-end
comparison misses entirely.

For a positive threshold the logic mirrors: it measures up from the period's *trough*.

**`period_to_period`** — compares the current close against the close from *n* days ago
and ignores everything in between. The same stock, if it started the week at $95, is
*down 1%* and would not fire a `-5%` alert.

```
        $100 ●
            ╱ ╲
     $95 ●╱     ╲● $94        from_peak:        -6.0%  ← fires at -5%
        └────────────         period_to_period: -1.1%  ← does not
      day 0        day 7
```

Use `from_peak` to catch drawdowns — including the ones that recover before you'd
otherwise have noticed. Use `period_to_period` when you care about net movement over a
fixed window.

### Period format

| Format | Unit | Examples |
|--------|------|----------|
| `Nd` | Days | `7d`, `30d`, `90d` |
| `Nw` | Weeks (7 days) | `1w`, `2w` |
| `Nm` | Months (30 days) | `1m`, `3m`, `6m` |
| `Ny` | Years (365 days) | `1y`, `2y` |

Months and years are approximations — `3m` is exactly 90 days, `1y` is exactly 365.

> **Free-tier limit:** periods longer than 120 days require Alpha Vantage's `full`
> history endpoint, which needs a **premium** key. Stick to `120d` or less on the free
> tier.

## Running on a schedule

The tool is stateless and exits when done, so any scheduler works.

### GitHub Actions (no machine required)

This is the easiest way to run it, and it costs nothing — scheduled workflows are free on
public repositories.

1. **Fork this repo.**
2. **Add your credentials as secrets.** Settings → Secrets and variables → Actions →
   *New repository secret*, three times:

   | Secret | Value |
   |--------|-------|
   | `ALPHAVANTAGE_API_KEY` | Your Alpha Vantage key |
   | `TELEGRAM_BOT_TOKEN` | Your bot token from @BotFather |
   | `TELEGRAM_CHAT_ID` | Your numeric chat ID, digits only |

3. **Edit `alerts.yaml`** in your fork and commit it. Unlike most config, this file *is*
   tracked — the workflow reads it out of the repo, so your watchlist has to be committed
   for hosted checks to work. Keep the fork private if you'd rather not publish it.
4. **Enable Actions** on the fork (the Actions tab asks once), and you're done.

[`.github/workflows/scheduled-check.yml`](.github/workflows/scheduled-check.yml) runs at
22:00 UTC on weekdays — 6pm ET during EDT, 5pm during EST, so it always lands after the
4pm close. Change the `cron:` line to suit your market. You can also trigger it by hand
from the Actions tab via **Run workflow**, which is the fastest way to confirm your
secrets are right.

> **Using an Actions environment instead?** If you put the secrets under an environment
> rather than at repository level, the job needs to name it — `environment: <name>` in the
> workflow. Without that, `secrets.*` come back empty and every run fails with
> `ALPHAVANTAGE_API_KEY is required`. This repo's own workflow points at an environment
> called `Prod`; delete that line if you use repository secrets.

> **Scheduled workflows go dormant.** GitHub disables cron triggers on public repos after
> 60 days without repository activity, and emails you when it does. Push a commit or
> re-enable it from the Actions tab.

### cron (Linux, macOS)

```bash
crontab -e
```

```cron
# Weekdays at 6 PM, after US market close
0 18 * * 1-5 cd /path/to/stock-drop-notifier && ./notifier check >> /tmp/notifier.log 2>&1
```

The `cd` matters — `.env` and `alerts.yaml` are resolved relative to the working
directory.

### launchd (macOS)

`make install` copies the binary to `~/bin/notifier` and your configs to
`~/.stock-notifier/`, then reloads a launchd agent if one exists at
`~/Library/LaunchAgents/com.user.stock-notifier.plist`. The repo does not ship a plist —
create one like this to run every weekday at 6 PM:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>              <string>com.user.stock-notifier</string>
    <key>ProgramArguments</key>
    <array>
        <string>/Users/YOUR_USERNAME/bin/notifier</string>
        <string>check</string>
    </array>
    <key>WorkingDirectory</key>   <string>/Users/YOUR_USERNAME/.stock-notifier</string>
    <key>StartCalendarInterval</key>
    <dict>
        <key>Hour</key>           <integer>18</integer>
        <key>Minute</key>         <integer>0</integer>
    </dict>
    <key>StandardOutPath</key>    <string>/tmp/stock-notifier.log</string>
    <key>StandardErrorPath</key>  <string>/tmp/stock-notifier.err</string>
</dict>
</plist>
```

```bash
launchctl load ~/Library/LaunchAgents/com.user.stock-notifier.plist
```

`make uninstall` removes the binary and unloads the agent, leaving your configs in place.

## Limits and caveats

**Alpha Vantage free tier: 25 requests/day, 5 requests/minute.** One alert costs one
request per `check` run, so 25 alerts checked once daily fits exactly. The tool spaces
requests 1.2 seconds apart automatically to stay inside the per-minute limit. Use
`--limit` if you need to cap a run.

**Daily closes only.** Alerts evaluate against end-of-day data, so an intraday crash
surfaces after the close, not as it happens.

**Some symbols need a premium key.** Index tickers like `IXIC` (NASDAQ Composite) and
`SPX` (S&P 500) are premium-only — note that `alerts.yaml.example` uses them for
illustration. Individual stocks and ETFs (`SPY`, `QQQ`, `VTI`) work fine on the free tier
and track the same indices closely. International tickers take a suffix, e.g. `TSCO.LON`.

**Config is read from the working directory.** Not from `$HOME`, not from a fixed path.

**Notifications are not deduplicated.** A stock that stays below your threshold sends a
message on every run. Widen the period, or disable the alert once you've seen it.

## Roadmap

- **Deduplicate repeat notifications across runs** — the next thing worth building. An
  alert currently re-fires on every run for as long as it stays triggered, so a sustained
  drawdown pings you daily. Firing only on the transition from OK to triggered needs a
  small amount of state carried between runs.
- Additional notification channels (email, webhooks, ntfy)
- Absolute price thresholds alongside percentage change

## Development

```bash
make build          # build ./notifier with version info
make test           # go test -v ./...
make test-coverage  # generate coverage.html
make lint           # golangci-lint (must be installed separately)
make tidy           # go mod tidy
make help           # list all targets
```

Layout:

```
cmd/notifier/          CLI entry point and command wiring (cobra)
internal/alphavantage/ API client, percent-change calculations
internal/alert/        Evaluation engine, alert types, message formatting
internal/config/       .env loading, alerts.yaml parsing and validation
internal/telegram/     Bot API client
```

## Contributing

Issues and pull requests are welcome. Please run `make test` and `make lint` before
opening a PR, and keep new behaviour covered by tests — the existing packages all have
test files worth following as examples.

## License

[MIT](LICENSE)
