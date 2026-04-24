package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/elvisoric/todo/store"
	"github.com/spf13/cobra"
)

var sortBy string

func init() {
	rootCmd.Flags().StringVarP(&sortBy, "sort", "o", "", "Sort order: due (by due date; no-due items last by creation)")
}

func sortTodos(todos []store.Todo, by string) error {
	switch by {
	case "":
		return nil
	case "due":
		sort.SliceStable(todos, func(i, j int) bool {
			a, b := todos[i], todos[j]
			switch {
			case a.DueAt != nil && b.DueAt != nil:
				return a.DueAt.Before(*b.DueAt)
			case a.DueAt != nil:
				return true
			case b.DueAt != nil:
				return false
			default:
				return a.CreatedAt.Before(b.CreatedAt)
			}
		})
		return nil
	default:
		return fmt.Errorf("unknown sort order %q (supported: due)", by)
	}
}

var rootCmd = &cobra.Command{
	Use:   "todo [space]",
	Short: "A terminal todo app with spaces",
	Long: `Manage todos organized by topics (spaces) with nested hierarchy.

Run 'todo' to see default todos and all spaces.
Run 'todo <space>' to see todos in that space (e.g. todo office).
Add '--sort=due' to any listing to order by due date (items without a due
date fall to the bottom, sorted by creation time).

Due dates:
  todo add "finish report" -d "tomorrow 2pm"
  todo due <id> "next monday"
  todo overdue              # items whose due date has passed
  todo alarm                # items due within a configured window (or overdue)

Configuration (YAML at $HOME/.todo/config.yaml, or $TODO_PATH):
  general:
    line_width: 120
    alarm_window: 1h

Shell Completion:
  Bash:  source <(todo completion bash)
  Zsh:   source <(todo completion zsh)
  Fish:  todo completion fish | source

  To make it permanent, add the command to your shell profile (~/.bashrc, ~/.zshrc, etc.).`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runList,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return completeSpaceNames(toComplete)
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(alarmCmd)
	rootCmd.AddCommand(doneCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(dueCmd)
	rootCmd.AddCommand(listDoneCmd)
	rootCmd.AddCommand(overdueCmd)
	rootCmd.AddCommand(spacesCmd)
	rootCmd.AddCommand(renameCmd)
	rootCmd.AddCommand(moveCmd)
	rootCmd.AddCommand(completionCmd)
	rootCmd.AddCommand(versionCmd)
}

// Default action: list active todos + space tree
func runList(cmd *cobra.Command, args []string) error {
	s, err := store.Load()
	if err != nil {
		return err
	}

	// If a space name is given, show todos in that space
	if len(args) == 1 {
		space := args[0]
		todos := s.TodosByNamespace(space)
		if len(todos) == 0 {
			fmt.Printf("No todos in space %q.\n", space)
			return nil
		}
		if err := sortTodos(todos, sortBy); err != nil {
			return err
		}
		fmt.Printf("Todos in %s:\n", c(boldCyan, space))
		printTodos(todos)
		return nil
	}

	// Show default todos
	var defaultTodos []store.Todo
	for _, t := range s.ActiveTodos() {
		if t.Namespace == "default" {
			defaultTodos = append(defaultTodos, t)
		}
	}

	if len(defaultTodos) == 0 {
		fmt.Println(c(dim, "No todos in default space."))
	} else {
		if err := sortTodos(defaultTodos, sortBy); err != nil {
			return err
		}
		renderList(todoRows(defaultTodos, false), getConfig().General.LineWidth)
	}

	// Show other spaces with counts
	tree := s.NamespaceTree()
	var hasOther bool
	for _, ch := range tree.Children {
		if ch.Name != "default" {
			hasOther = true
			break
		}
	}
	if hasOther {
		fmt.Println()
		fmt.Println(c(bold, "Spaces:"))
		for _, ch := range tree.Children {
			if ch.Name != "default" {
				printTree(ch, "  ")
			}
		}
	}

	return nil
}

func printTree(n *store.NamespaceNode, indent string) {
	if n.Name == "root" {
		for _, ch := range n.Children {
			printTree(ch, indent)
		}
		return
	}
	fmt.Printf("%s%s %s\n", indent, c(cyan, n.Name), c(dim, fmt.Sprintf("(%d)", n.Total)))
	for _, ch := range n.Children {
		printTree(ch, indent+"  ")
	}
}

func printTodos(todos []store.Todo) {
	if len(todos) == 0 {
		return
	}
	renderList(todoRows(todos, true), getConfig().General.LineWidth)
}

func todoRows(todos []store.Todo, showNS bool) []listRow {
	now := time.Now()
	rows := make([]listRow, 0, len(todos))
	for _, t := range todos {
		r := listRow{
			ID:      c(yellow, "["+t.ID+"]"),
			Text:    t.Text,
			Due:     dueCell(t, now),
			Created: c(dim, fmtTime(t.CreatedAt)),
		}
		if showNS {
			r.NS = c(magenta, t.Namespace)
		}
		if t.DoneAt != nil {
			r.Done = c(green, "(done "+fmtTime(*t.DoneAt)+")")
		}
		rows = append(rows, r)
	}
	return rows
}

func dueCell(t store.Todo, now time.Time) string {
	if t.DueAt == nil {
		return ""
	}
	label, color := formatDue(*t.DueAt, now)
	return c(color, label)
}

func fmtTime(t time.Time) string {
	return t.Format("2006-01-02 15:04")
}

// --- add ---
var (
	addNamespace string
	addDue       string
)

var addCmd = &cobra.Command{
	Use:   "add [text]",
	Short: "Add a new todo",
	Long: `Add a new todo. Use -d to set a due date.

Due-date examples:
  todo add "call mom" -d "tomorrow 2pm"
  todo add "finish report" -d "today 16:00"
  todo add "write letter" -d "next monday"
  todo add "ship feature" -d "2026-04-30 14:00"
  todo add "drink water" -d "in 2h"
  todo add "weekly sync" -d friday`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		text := strings.Join(args, " ")
		var due *time.Time
		if addDue != "" {
			t, err := ParseDue(addDue)
			if err != nil {
				return err
			}
			due = &t
		}
		s, err := store.Load()
		if err != nil {
			return err
		}
		t := s.Add(text, addNamespace, due)
		if err := s.Save(); err != nil {
			return err
		}
		msg := fmt.Sprintf("%s %s to %s: %s", c(boldGreen, "+"), c(yellow, "["+t.ID+"]"), c(cyan, t.Namespace), t.Text)
		if due != nil {
			label, color := formatDue(*due, time.Now())
			msg += " " + c(color, "("+label+")")
		}
		fmt.Println(msg)
		return nil
	},
}

func init() {
	addCmd.Flags().StringVarP(&addNamespace, "space", "s", "", "Topic/space (dot-separated, e.g. office.monday)")
	addCmd.Flags().StringVarP(&addDue, "due", "d", "", "Due date (e.g. \"tomorrow 2pm\", \"2026-04-30\", \"next monday\", \"in 3h\")")
}

// --- done ---
var doneCmd = &cobra.Command{
	Use:   "done [id...]",
	Short: "Mark todos as done",
	Args:  cobra.MinimumNArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return completeTodoIDs(toComplete, false)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := store.Load()
		if err != nil {
			return err
		}
		done, notFound := s.MarkDone(args)
		if err := s.Save(); err != nil {
			return err
		}
		for _, id := range done {
			fmt.Printf("%s %s marked as done.\n", c(boldGreen, "✓"), c(yellow, "["+id+"]"))
		}
		for _, id := range notFound {
			fmt.Fprintf(os.Stderr, "%s Todo %s not found.\n", c(boldRed, "✗"), c(yellow, "["+id+"]"))
		}
		return nil
	},
}

// --- delete ---
var deleteCmd = &cobra.Command{
	Use:   "delete [id...]",
	Short: "Delete todos",
	Args:  cobra.MinimumNArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return completeTodoIDs(toComplete, true)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := store.Load()
		if err != nil {
			return err
		}
		deleted, notFound := s.Delete(args)
		if err := s.Save(); err != nil {
			return err
		}
		for _, id := range deleted {
			fmt.Printf("%s %s deleted.\n", c(boldRed, "−"), c(yellow, "["+id+"]"))
		}
		for _, id := range notFound {
			fmt.Fprintf(os.Stderr, "%s Todo %s not found.\n", c(boldRed, "✗"), c(yellow, "["+id+"]"))
		}
		return nil
	},
}

// --- move ---
var moveSpace string

var moveCmd = &cobra.Command{
	Use:   "move [id...] -s [space]",
	Short: "Move todos to a different space",
	Args:  cobra.MinimumNArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return completeTodoIDs(toComplete, false)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if moveSpace == "" {
			return fmt.Errorf("target space required: use -s <space>")
		}
		s, err := store.Load()
		if err != nil {
			return err
		}
		moved, notFound := s.MoveTodos(args, moveSpace)
		if err := s.Save(); err != nil {
			return err
		}
		for _, id := range moved {
			fmt.Printf("%s %s moved to %s.\n", c(boldGreen, "→"), c(yellow, "["+id+"]"), c(cyan, moveSpace))
		}
		for _, id := range notFound {
			fmt.Fprintf(os.Stderr, "%s Todo %s not found.\n", c(boldRed, "✗"), c(yellow, "["+id+"]"))
		}
		return nil
	},
}

func init() {
	moveCmd.Flags().StringVarP(&moveSpace, "space", "s", "", "Target space to move todos to")
	moveCmd.MarkFlagRequired("space")
}

// --- rename ---
var renameCmd = &cobra.Command{
	Use:   "rename [old-space] [new-space]",
	Short: "Rename a space",
	Long:  "Rename a space and all its children. E.g. 'todo rename office work' renames office.* to work.*",
	Args:  cobra.ExactArgs(2),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) >= 2 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return completeSpaceNames(toComplete)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		oldName, newName := args[0], args[1]
		s, err := store.Load()
		if err != nil {
			return err
		}
		count := s.RenameSpace(oldName, newName)
		if count == 0 {
			fmt.Fprintf(os.Stderr, "%s No todos found in space %s.\n", c(boldRed, "✗"), c(cyan, oldName))
			return nil
		}
		if err := s.Save(); err != nil {
			return err
		}
		fmt.Printf("%s Renamed %s to %s (%d todos updated).\n", c(boldGreen, "✓"), c(cyan, oldName), c(cyan, newName), count)
		return nil
	},
}

// --- spaces ---
var spacesCmd = &cobra.Command{
	Use:   "spaces",
	Short: "List all spaces with todo counts",
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := store.Load()
		if err != nil {
			return err
		}
		tree := s.NamespaceTree()
		if tree.Total == 0 {
			fmt.Println(c(dim, "No active todos in any space."))
			return nil
		}
		for _, ch := range tree.Children {
			printTree(ch, "")
		}
		return nil
	},
}

// --- due ---
var dueClear bool

var dueCmd = &cobra.Command{
	Use:   "due [id] [date...]",
	Short: "Set or clear the due date of a todo",
	Long: `Set the due date of an existing todo.

Examples:
  todo due abcd "next friday"
  todo due abcd tomorrow 2pm
  todo due abcd 2026-04-30 14:00
  todo due abcd --clear`,
	Args: func(cmd *cobra.Command, args []string) error {
		if dueClear {
			if len(args) != 1 {
				return fmt.Errorf("--clear requires exactly one id")
			}
			return nil
		}
		if len(args) < 2 {
			return fmt.Errorf("expected <id> and a date (or use --clear)")
		}
		return nil
	},
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return completeTodoIDs(toComplete, false)
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		s, err := store.Load()
		if err != nil {
			return err
		}
		t := s.FindByID(id)
		if t == nil {
			return fmt.Errorf("todo %s not found", id)
		}
		if dueClear {
			t.DueAt = nil
			if err := s.Save(); err != nil {
				return err
			}
			fmt.Printf("%s %s due date cleared.\n", c(boldGreen, "✓"), c(yellow, "["+t.ID+"]"))
			return nil
		}
		expr := strings.Join(args[1:], " ")
		when, err := ParseDue(expr)
		if err != nil {
			return err
		}
		t.DueAt = &when
		if err := s.Save(); err != nil {
			return err
		}
		label, color := formatDue(when, time.Now())
		fmt.Printf("%s %s %s\n", c(boldGreen, "✓"), c(yellow, "["+t.ID+"]"), c(color, label))
		return nil
	},
}

func init() {
	dueCmd.Flags().BoolVar(&dueClear, "clear", false, "Clear the due date instead of setting it")
}

// --- overdue ---
var overdueAll bool

var overdueCmd = &cobra.Command{
	Use:   "overdue",
	Short: "List todos whose due date has passed",
	Long:  "List active todos whose due date has passed. Defaults to the default space; use --all for every space.",
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := store.Load()
		if err != nil {
			return err
		}
		now := time.Now()
		var overdue []store.Todo
		for _, t := range s.ActiveTodos() {
			if t.DueAt == nil || !t.DueAt.Before(now) {
				continue
			}
			if !overdueAll && t.Namespace != "default" {
				continue
			}
			overdue = append(overdue, t)
		}
		if len(overdue) == 0 {
			fmt.Println(c(dim, "No overdue todos."))
			return nil
		}
		sort.SliceStable(overdue, func(i, j int) bool {
			return overdue[i].DueAt.Before(*overdue[j].DueAt)
		})
		if overdueAll {
			printTodos(overdue)
			return nil
		}
		renderList(todoRows(overdue, false), getConfig().General.LineWidth)
		return nil
	},
}

func init() {
	overdueCmd.Flags().BoolVar(&overdueAll, "all", false, "Include overdue todos from every space, not just default")
}

// --- list-done ---
var listDoneCmd = &cobra.Command{
	Use:   "list-done",
	Short: "List completed todos",
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := store.Load()
		if err != nil {
			return err
		}
		todos := s.DoneTodos()
		if len(todos) == 0 {
			fmt.Println(c(dim, "No completed todos."))
			return nil
		}
		fmt.Println(c(bold, "Completed todos:"))
		printTodos(todos)
		return nil
	},
}

// --- completion ---
var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish]",
	Short: "Generate shell completion scripts",
	Long: `Generate shell completion scripts for your shell.

To load completions:

Bash:
  $ source <(todo completion bash)
  # Or add to ~/.bashrc:
  $ echo 'source <(todo completion bash)' >> ~/.bashrc

Zsh:
  $ source <(todo completion zsh)
  # Or add to ~/.zshrc:
  $ echo 'source <(todo completion zsh)' >> ~/.zshrc

Fish:
  $ todo completion fish | source
  # Or persist:
  $ todo completion fish > ~/.config/fish/completions/todo.fish
`,
	Args:      cobra.ExactArgs(1),
	ValidArgs: []string{"bash", "zsh", "fish"},
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return rootCmd.GenBashCompletionV2(os.Stdout, true)
		case "zsh":
			return rootCmd.GenZshCompletion(os.Stdout)
		case "fish":
			return rootCmd.GenFishCompletion(os.Stdout, true)
		default:
			return fmt.Errorf("unsupported shell: %s", args[0])
		}
	},
}

func completeTodoIDs(toComplete string, includeAll bool) ([]string, cobra.ShellCompDirective) {
	s, err := store.Load()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	var completions []string
	var todos []store.Todo
	if includeAll {
		todos = s.Todos
	} else {
		todos = s.ActiveTodos()
	}
	for _, t := range todos {
		if strings.HasPrefix(t.ID, strings.ToLower(toComplete)) {
			completions = append(completions, fmt.Sprintf("%s\t[%s] %s", t.ID, t.Namespace, store.Truncate(t.Text, 40)))
		}
	}
	return completions, cobra.ShellCompDirectiveNoFileComp
}

func completeSpaceNames(toComplete string) ([]string, cobra.ShellCompDirective) {
	s, err := store.Load()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	var completions []string
	for _, ns := range s.AllNamespaces() {
		if strings.HasPrefix(ns, toComplete) {
			completions = append(completions, ns)
		}
	}
	return completions, cobra.ShellCompDirectiveNoFileComp
}
