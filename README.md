# claude-code-powerline

A fast, single-binary statusline for [Claude Code](https://claude.ai/claude-code). Replaces shell-script + starship hacks with a native Go binary that reads Claude Code's statusline JSON from stdin and outputs a formatted powerline.

Zero subprocesses for rendering (only `git status --porcelain` for dirty detection).

## Install

```bash
go install github.com/tylergannon/claude-code-powerline@latest
```

## Configure

In your Claude Code settings, set the statusline command:

```json
{
  "statusLine": {
    "type": "command",
    "command": "claude-code-powerline"
  }
}
```

## Segments

| Segment | Description |
|---------|-------------|
| Context battery | 5-cell bar, green/yellow/red by usage. Shows `Nk/Nk` when >= 10% used. |
| Rate limits | Shows `5h:N%` / `7d:N%` when either exceeds 50%. Red at 80%+. |
| Model | Dimmed model display name. |
| Directory | Current working dir, `~`-shortened, truncated to 3 components. |
| Git | Branch name (read from `.git/HEAD`, no subprocess) + `✎` if dirty. |
| Cost | Session cost in USD, shown when >= $0.01. |

## Why not starship?

The typical starship-based statusline spawns 15-18 processes per render (shell, jq, starship, custom module subshells, git). This binary spawns 1 process (itself) plus 1 `git status` call. On a fast machine the difference is negligible, but over a long session the process churn from the shell approach can leak handles.
