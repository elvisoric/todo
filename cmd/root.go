package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/elvisoric/todo/store"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "todo [space]",
	Short: "A terminal todo app with spaces",
	Long:  "Manage todos organized by topics (spaces) with nested hierarchy.\n\nRun 'todo' to see default todos and all spaces.\nRun 'todo <space>' to see todos in that space (e.g. todo office).",
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
	rootCmd.AddCommand(doneCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(listDoneCmd)
	rootCmd.AddCommand(spacesCmd)
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
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		for _, t := range defaultTodos {
			fmt.Fprintf(w, "  %s\t%s\t%s\n", c(yellow, "["+t.ID+"]"), t.Text, c(dim, fmtTime(t.CreatedAt)))
		}
		w.Flush()
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

func printTodosByNamespace(s *store.Store, ns, label string) {
	var todos []store.Todo
	for _, t := range s.ActiveTodos() {
		if t.Namespace == ns {
			todos = append(todos, t)
		}
	}
	if len(todos) == 0 {
		return
	}
	fmt.Printf("%s %s %s\n", c(dim, "──"), c(boldCyan, label), c(dim, "──"))
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for _, t := range todos {
		fmt.Fprintf(w, "  %s\t%s\t%s\n", c(yellow, "["+t.ID+"]"), t.Text, c(dim, fmtTime(t.CreatedAt)))
	}
	w.Flush()
	fmt.Println()
}

func printTodos(todos []store.Todo) {
	if len(todos) == 0 {
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for _, t := range todos {
		ts := c(dim, fmtTime(t.CreatedAt))
		doneTS := ""
		if t.DoneAt != nil {
			doneTS = c(green, " (done "+fmtTime(*t.DoneAt)+")")
		}
		fmt.Fprintf(w, "  %s\t%s\t%s\t%s%s\n", c(yellow, "["+t.ID+"]"), c(magenta, t.Namespace), t.Text, ts, doneTS)
	}
	w.Flush()
}

func fmtTime(t time.Time) string {
	return t.Format("2006-01-02 15:04")
}

// --- add ---
var addNamespace string

var addCmd = &cobra.Command{
	Use:   "add [text]",
	Short: "Add a new todo",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		text := strings.Join(args, " ")
		s, err := store.Load()
		if err != nil {
			return err
		}
		t := s.Add(text, addNamespace)
		if err := s.Save(); err != nil {
			return err
		}
		fmt.Printf("%s %s to %s: %s\n", c(boldGreen, "+"), c(yellow, "["+t.ID+"]"), c(cyan, t.Namespace), t.Text)
		return nil
	},
}

func init() {
	addCmd.Flags().StringVarP(&addNamespace, "space", "s", "", "Topic/space (dot-separated, e.g. office.monday)")
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
