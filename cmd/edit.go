package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/elvisoric/todo/store"
	"github.com/spf13/cobra"
)

var editCmd = &cobra.Command{
	Use:   "edit [id] [text...]",
	Short: "Edit a todo's text",
	Long: `Edit a todo's text.

With text on the command line, the text is replaced directly:
  todo edit abcd "updated text for this todo"

Without text, $EDITOR (falls back to $VISUAL, then nano/vim/vi) opens with
the current text pre-filled:
  todo edit abcd`,
	Args: cobra.MinimumNArgs(1),
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

		var newText string
		if len(args) > 1 {
			newText = strings.Join(args[1:], " ")
		} else {
			edited, err := openInEditor(t.Text)
			if err != nil {
				return err
			}
			newText = edited
		}

		newText = normalizeTodoText(newText)
		if newText == "" {
			return fmt.Errorf("todo text cannot be empty (use 'todo delete %s' to remove)", t.ID)
		}
		if newText == t.Text {
			fmt.Printf("%s %s no changes.\n", c(dim, "·"), c(yellow, "["+t.ID+"]"))
			return nil
		}

		t.Text = newText
		if err := s.Save(); err != nil {
			return err
		}
		fmt.Printf("%s %s %s\n", c(boldGreen, "✓"), c(yellow, "["+t.ID+"]"), t.Text)
		return nil
	},
}

func normalizeTodoText(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.TrimSpace(s)
}

func openInEditor(initial string) (string, error) {
	editor := os.Getenv("VISUAL")
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if editor == "" {
		for _, cand := range []string{"nano", "vim", "vi"} {
			if _, err := exec.LookPath(cand); err == nil {
				editor = cand
				break
			}
		}
	}
	if editor == "" {
		return "", fmt.Errorf("no editor found: set $EDITOR (or $VISUAL)")
	}

	f, err := os.CreateTemp("", "todo-edit-*.txt")
	if err != nil {
		return "", err
	}
	tmp := f.Name()
	defer os.Remove(tmp)

	if _, err := f.WriteString(initial); err != nil {
		f.Close()
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}

	parts := strings.Fields(editor)
	parts = append(parts, tmp)
	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("editor %q failed: %w", editor, err)
	}

	data, err := os.ReadFile(tmp)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
