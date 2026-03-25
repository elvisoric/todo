# todo

A fast, minimal terminal todo app with nested spaces (namespaces) written in Go.

Organize your todos into dot-separated spaces like `office.monday.morning` and browse them from the command line.

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
```

### List todos

```bash
# Show default space todos + summary of other spaces
todo

# Show todos in a specific space (includes children)
todo office
todo office.monday
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

## Todo IDs

Each todo gets a unique 4-character hex ID (e.g., `a1b2`, `f0e9`). Use these IDs with `done` and `delete` commands.

## Storage

Todos are stored as JSON in `~/.todo/todos.json`. No database required.

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

Output is color-coded for readability. Colors are automatically disabled when piping output or when the `NO_COLOR` environment variable is set.

## License

MIT
