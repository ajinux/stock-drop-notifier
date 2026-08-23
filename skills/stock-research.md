# Stock research skill

Instructions for an AI coding assistant (Claude Code, Cursor, Aider, Codex CLI, or
similar — anything with web browsing, a shell, and file-edit access) to help a user of
this repo find stocks worth watching and add them to `alerts.yaml`.

This is a *guided, on-demand* workflow. It never runs unattended, and it never pushes
without an explicit go-ahead — the person driving reviews every ticker before it's
written to disk, and reviews the diff before it leaves their machine.

## When to use this

The user asks something like "what stocks should I watch", "find me some volatile tech
names", "suggest something to add to my alerts", or points you at this file directly.

## Steps

### 1. Scope the research

If the user already gave a focus (a sector, specific tickers, "broad market movers"), use
it. Otherwise ask briefly rather than guessing.

Either way, read `alerts.yaml` first:

- Don't suggest a symbol that's already in there.
- If the user gave no focus, use the existing alerts as a hint — sector mix, index funds
  vs. single names, typical thresholds — so suggestions fit the rest of the watchlist
  instead of feeling random.

### 2. Research candidates on the web

Use whatever search/browsing capability you have. Alpha Vantage — this project's only
market-data source — has no news or screening endpoint on its free tier, so discovery has
to come from general web search, not the `notifier` CLI or the Go client in
`internal/alphavantage/`.

Look for a *concrete* reason to watch each candidate: recent volatility, an upcoming
catalyst (earnings date, product launch), a notable drawdown from a recent high, or plain
relevance to the requested sector. "It's a big company" is not a reason.

### 3. Present a shortlist

Around 3–7 candidates, as a table:

| Symbol | Why watch it | Proposed threshold | Proposed period | Method |
|---|---|---|---|---|

Base the proposed threshold/period/method on what's already in `alerts.yaml` rather than
inventing new conventions per stock — this repo's existing alerts mostly use drops of
-5% to -10% over 1–3 months with the `from_peak` calculation method (see
`internal/config/alerts.go` for what these fields mean if the file isn't self-explanatory).

### 4. Get explicit approval

Let the user accept some, reject others, or adjust a threshold/period, before writing
anything. Don't add a symbol they didn't approve.

### 5. Add approved picks with `notifier add`

Use the existing CLI command rather than hand-editing the YAML — it validates the period
format and rejects duplicate names for you:

```bash
go run ./cmd/notifier add --symbol NVDA --threshold -8 --period 2m --name "NVDA 2-Month Drop"
```

One call per approved symbol.

### 6. Sanity-check the new symbols, quota permitting

Alpha Vantage's free tier is 25 requests/day. Don't run a full `notifier check` — that
burns a request per *existing* alert too. Instead, if the day's quota clearly allows it,
check just the newly-added ones, and skip this step entirely rather than guess at the
remaining quota.

A `SymbolNotFoundError` means the ticker was wrong — `notifier remove` it and let the
user know, rather than leaving a broken alert in the file.

### 7. Show the diff and commit — then stop

```bash
git diff alerts.yaml
```

Commit with a plain, descriptive message — no attribution trailer, no tool-credit line.

**Then stop.** Tell the user what was committed and ask before running `git push`. Never
push without that explicit go-ahead, even if they approved the stock picks earlier —
approving tickers is not approval to push.
