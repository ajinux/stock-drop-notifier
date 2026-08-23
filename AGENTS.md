# Agent Instructions: Stock Drop Notifier

These instructions are specifically tailored for future AI agents working in this repository to prevent common mistakes, understand system assumptions, and verify work correctly.

If you're being asked to help find stocks worth watching (not just manage existing alerts), follow `skills/stock-research.md` instead of improvising — it covers scoping, sourcing, and the approval/commit/push sequence this repo expects. In Claude Code, this is also registered as an invokable skill (`.claude/skills/stock-research/SKILL.md`, runs via `/stock-research`) — that file just points back here, so `skills/stock-research.md` remains the one canonical version to edit.

---

## ⚡ Core Domain Constraints & Assumptions

### 1. Alpha Vantage API Rate Limiting
* **Limits:** Free tier allows **25 requests/day** and **5 requests/minute**.
* **Enforced Delay:** The engine enforces a **1.2-second sleep delay (`1200ms`)** between consecutive ticker evaluations.
* **Testing:** In tests, ALWAYS disable this delay to prevent extremely slow suites by calling `.WithRateLimitDelay(0)` on the Engine.

### 2. Time Series Data Boundaries (Compact vs. Full)
* **Threshold (120 days):** Time periods $\le 120$ days request `compact` output (fast, ~100 trading days). Periods $> 120$ days automatically request `full` output size.
* **Premium Restriction:** Requests for `full` size (and any index symbol like `IXIC` or `SPX`) will return a rate-limit or error if using a standard free API key. The parser detects this and throws a `PremiumRequiredError`.

### 3. Calculation Methods (`calculation_method` field)
There are two supported calculation methods:
1. **`from_peak` (Default):** Calculates drop from the peak (maximum close) within the period to current price, or gain from the trough (minimum close) within the period to current price.
2. **`period_to_period`:** Compares current price to the price exactly $N$ days ago.

### 4. Period Formats
Supported units are `d` (days), `w` (weeks = 7d), `m` (months = 30d), and `y` (years = 365d). Parsing is handled via `config.ParsePeriod()`.

---

## 🛠️ Developer Commands & Automation

### 1. Build and Test
```bash
make build          # Compiles standard binary to ./notifier
make test           # Runs full test suite (33+ tests)
make test-coverage  # Generates test coverage HTML report
```

### 2. Launchd & Deployment Automation
The project uses a custom launchd workflow for local automation on macOS.
* **`make install`**:
  1. Compiles a fresh binary to `~/bin/notifier`.
  2. Copies `.env` and `alerts.yaml` directly to `~/.stock-notifier/` (overwrites directly).
  3. Reloads the launchd plist agent if present at `~/Library/LaunchAgents/com.user.stock-notifier.plist`. Fails hard if plist load fails.
* **`make uninstall`**:
  1. Stops/unloads the launchd agent.
  2. Removes binary but preserves `.env` and `alerts.yaml` settings.

### 3. Timezone Schedule (IST to US ET Mapping)
* The launchd configuration runs at **2:00 AM IST** (Tuesday through Saturday mornings).
* This maps to **Monday through Friday ET** post-market-close (NYSE close at 4:00 PM ET is 1:30 AM IST).
* Sunday and Monday mornings (IST) are omitted as the US market is closed on weekends.

---

## 🔍 Verification Checklist

Before pushing modifications or considering a task done, verify:

1. **Check command behaves correctly with limits:**
   * `--limit <n>` / `-l <n>` restricts evaluation to the first $n$ alerts.
   * Negative values are strictly validated and fail with `limit must be a non-negative integer`.
2. **Launchd path requirements:**
   * The plist must use absolute paths — launchd does not expand `~`.
   * The agent runs with working directory `~/.stock-notifier` and invokes `~/bin/notifier`,
     since `.env` and `alerts.yaml` are resolved relative to the working directory.
3. **No configurations are committed:**
   * Never commit absolute directories, `.env`, or personal keys. Always write to `~/.stock-notifier/` for active local cron-like checks.
