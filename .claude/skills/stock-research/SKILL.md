---
name: stock-research
description: "Use when the user asks what stocks to watch, wants volatile/watchlist candidates, or asks to suggest additions to alerts.yaml in this repo. Guided, on-demand workflow: read alerts.yaml, research candidates on the web with concrete reasons, present a shortlist, get explicit approval, add approved picks with `notifier add`, sanity-check quota permitting, then show the diff and commit (never push without a separate go-ahead)."
---

Follow the full instructions in `skills/stock-research.md` at the root of this
repository, exactly as written. That file is the canonical, tool-agnostic
version of this skill (also read by Cursor, Aider, Codex CLI, etc.), so this
SKILL.md exists only to make it discoverable/invokable in Claude Code — do not
duplicate or fork its content here. If the two ever diverge, treat
`skills/stock-research.md` as the source of truth and update this pointer
file instead of drifting from it.
