# todo

A fast, minimal terminal todo app with nested spaces (namespaces), due dates,
and an alarm view for what's coming up soon — written in Go.

Organize your todos into dot-separated spaces like `office.monday.morning`, give
them natural-language due dates (`tomorrow 2pm`, `next monday`, `in 3h`), and
browse them from the command line.

## Install

```bash
go install github.com/elvisoric/todo@latest
```

Make sure `$GOPATH/bin` is in your `PATH`. Add this to your `~/.zshrc` (or `~/.bashrc`):

```bash
export PATH="$PATH:$(go env GOPATH)/bin"
```

Then restart your shell or run `source ~/.zshrc`.

Or build from source:

```bash
git clone https://github.com/elvisoric/todo.git
cd todo
go build -o todo .
# Move the binary somewhere in your PATH, e.g.:
mv todo /usr/local/bin/
```

## Usage

### Add todos

```bash
# Add to default space
todo add "Fix the login bug"

# Add to a space
todo add -s office "Prepare meeting slides"

# Spaces can be nested with dots
todo add -s office.monday.morning "Review weekly report"

# Add with a due date
todo add "Call mom" -d "tomorrow 2pm"
todo add "Ship feature" -d "2026-04-30 14:00" -s work
todo add "Drink water" -d "in 2h"
todo add "Weekly sync" -d "next monday"
```

### List todos

```bash
# Show default space todos + summary of other spaces
todo

# Show todos in a specific space (includes children)
todo office
todo office.monday

# Sort by due date (items without a due date fall to the bottom)
todo --sort=due
todo office -o due
```

### List all spaces

```bash
todo spaces
```

Output:
```
default (2)
home (1)
office (3)
  monday (2)
    morning (1)
```

### Due dates

Set, change, or clear the due date of an existing todo:

```bash
todo due a1b2 "next friday"
todo due a1b2 tomorrow 2pm
todo due a1b2 2026-04-30 14:00
todo due a1b2 --clear
```

Supported date expressions include absolute dates (`2026-04-30`,
`2026-04-30 14:00`, `30.04.2026`), keywords (`today`, `tomorrow`, `tonight`,
`yesterday`), weekday names (`monday`, `next friday`), bare times (`2pm`,
`14:00` — bumps to tomorrow if already past), and relative durations
(`in 2h`, `in 30 minutes`, `+3d`, `2h`).

Due dates show up inline in every listing, colored by urgency:

```
  [43f4]  call mom       due tomorrow 14:00  2026-04-24 06:35
  [5080]  drink water    due today 08:35     2026-04-24 06:35
  [bfec]  send contract  overdue 1d ago      2026-04-24 06:35
```

### Overdue

List active todos whose due date has already passed:

```bash
# Default space only
todo overdue

# All spaces
todo overdue --all
```

### Alarm (upcoming + overdue)

List todos that are overdue or due within a time window. The window defaults to
`general.alarm_window` from the config (1h if unset), and can be overridden
per-invocation:

```bash
todo alarm                  # defaults to the configured window
todo alarm -w 30m           # anything due within the next 30 minutes (or overdue)
todo alarm -s work          # scope to one space
todo alarm -f plain         # machine-friendly one-per-line output, no colors
```

The `plain` format composes nicely with other tools:

```bash
# Send each alarm to notify-send (Linux)
todo alarm -f plain | while read -r l; do notify-send "todo" "$l"; done

# Cron: check every 5 minutes with a 30-minute lookahead
*/5 * * * * todo alarm -f plain -w 30m
```

### Edit a todo

Replace the text of an existing todo. With text on the command line it's a
direct replacement; with no text, `$EDITOR` (falls back to `$VISUAL`, then
`nano`/`vim`/`vi`) opens pre-filled with the current text.

```bash
todo edit a1b2 "new text for this todo"
todo edit a1b2
```

### Mark as done

```bash
# Single
todo done a1b2

# Multiple
todo done a1b2 c3d4 e5f6
```

### List completed todos

```bash
todo list-done
```

### Delete todos

```bash
todo delete a1b2
todo delete a1b2 c3d4
```

### Move todos

Move one or more todos to a different space:

```bash
todo move a1b2 -s work
todo move a1b2 c3d4 -s office.monday
```

### Rename a space

Rename a space (and all its children):

```bash
# Renames office.* to work.*
todo rename office work
```

## Todo IDs

Each todo gets a unique 4-character hex ID (e.g., `a1b2`, `f0e9`). Use these IDs
with `done`, `delete`, `move`, and `due` commands.

## Configuration

Optional YAML config. `todo` looks for it at `$TODO_PATH` if set, otherwise
`$HOME/.todo/config.yaml` (same directory as your todos). If the file is
missing or malformed, built-in defaults apply.

```yaml
general:
  line_width: 120     # wrap the text column to this many characters (default 120)
  alarm_window: 1h    # default window for 'todo alarm'; accepts 30m, 1h, 1h30m, 2d, 1w
```

Long todo text wraps at the configured `line_width`; continuation lines
align under the text column so the table stays readable.

## Storage

Todos are stored as JSON in `~/.todo/todos.json`. No database required. The
optional config file lives in the same directory
(see [Configuration](#configuration)).

## Shell Completion

```bash
# Bash
echo 'source <(todo completion bash)' >> ~/.bashrc

# Zsh
echo 'source <(todo completion zsh)' >> ~/.zshrc

# Fish
todo completion fish > ~/.config/fish/completions/todo.fish
```

Completions include subcommands, space names, and todo IDs with text previews.

## Colors

Output is color-coded for readability (e.g., overdue todos in red, due-soon in
yellow). Colors are automatically disabled when piping output or when the
`NO_COLOR` environment variable is set.

## License

MIT
