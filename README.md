# claude-code-powerline

A fast, single-binary statusline for [Claude Code](https://claude.ai/claude-code).

```bash
go install github.com/tylergannon/claude-code-powerline@latest
```

Then add to `~/.claude/settings.json`:

```json
{
  "statusLine": {
    "type": "command",
    "command": "claude-code-powerline"
  }
}
```

Restart Claude Code and you're done.

---

Replaces shell-script + starship hacks with a native Go binary that parses Claude Code's JSON on stdin and outputs a compact, color-coded powerline. One process, no dependencies.

![segments illustration](https://img.shields.io/badge/context-battery-green) ![segments illustration](https://img.shields.io/badge/rate_limits-yellow-yellow) ![segments illustration](https://img.shields.io/badge/model-white-lightgrey) ![segments illustration](https://img.shields.io/badge/directory-cyan-cyan) ![segments illustration](https://img.shields.io/badge/git-purple-purple) ![segments illustration](https://img.shields.io/badge/worktree-yellow-yellow) ![segments illustration](https://img.shields.io/badge/cost-white-lightgrey)

```
██░░░ 60k/200k (30%) │ 7d:55% │ Opus │ ~/src/my-project │  main ✎ │ ⌥ my-worktree │ $0.15
```

## Prerequisites

- **Go 1.21+** — [install Go](https://go.dev/dl/)
- **Claude Code** — the CLI, desktop app, or IDE extension

Make sure `$GOPATH/bin` (usually `~/go/bin`) is on your `PATH`. If you're not sure:

```bash
# Add to your shell profile (~/.bashrc, ~/.zshrc, or ~/.config/fish/config.fish)
export PATH="$PATH:$(go env GOPATH)/bin"
```

Verify it works:

```bash
echo '{"context_window":{"context_window_size":200000,"used_percentage":30}}' | claude-code-powerline
```

You should see a green battery bar.

## What you get

| Segment | What it shows | When it appears |
|---------|---------------|-----------------|
| **Context battery** | `██░░░ 60k/200k (30%)` — 5-cell bar with token counts and percentage | Always (green < 50%, yellow 50-79%, bold red 80%+) |
| **Rate limits** | `5h:82% 7d:91%` — your Claude usage limits | Only when a limit exceeds 50% |
| **Model** | `Opus`, `Sonnet`, etc. | Always |
| **Directory** | `~/src/my-project` — home-shortened, truncated to 3 components | Always |
| **Git** | ` main ✎` — branch from `.git/HEAD` + dirty indicator | Only in git repos |
| **Worktree** | `⌥ my-worktree` — active worktree name | Only when in an isolated worktree |
| **Cost** | `$0.15` — session cost in USD | When >= $0.01 |

### Color thresholds

- **Context battery**: green (< 50%) → yellow (50-79%) → bold red (80%+)
- **Rate limits**: yellow (50-79%) → bold red (80%+)
- Segments that aren't relevant (no git repo, no rate limit pressure, low cost) are hidden automatically.

## Why this instead of starship?

A starship-based statusline spawns **15-18 processes per render**: the shell interpreter, `cat`, multiple `jq` pipes, the `starship` binary itself, subshells for each custom module's `when` and `command`, plus `git` calls. Over a long Claude Code session, these accumulate and can leak process handles.

This binary spawns **1 process** (itself) plus 1 `git status --porcelain` call for dirty detection. Git branch is read directly from `.git/HEAD` with no subprocess.

## Uninstall

```bash
rm "$(go env GOPATH)/bin/claude-code-powerline"
```

Remove the `"statusLine"` block from `~/.claude/settings.json`.

## Credits

Thanks to [@jtokash](https://github.com/jtokash) whose bash + starship statusline script was the starting point for this tool.

## License

MIT
