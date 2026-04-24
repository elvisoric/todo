package cmd

import (
	"fmt"
	"sort"
	"time"

	"github.com/elvisoric/todo/store"
	"github.com/spf13/cobra"
)

var (
	alarmWithin string
	alarmFormat string
	alarmSpace  string
)

var alarmCmd = &cobra.Command{
	Use:   "alarm",
	Short: "Print todos that are overdue or due within a window",
	Long: `Print active todos that are overdue or due within a time window.
Defaults the window to the configured general.alarm_window (1h if unset).

Compose with other tools:
  todo alarm                                 # pretty output in a terminal
  todo alarm -f plain                        # one line per todo, no colors
  todo alarm -f plain | while read -r l; do notify-send "todo" "$l"; done
  */5 * * * * todo alarm -f plain -w 30m     # cron`,
	RunE: func(cmd *cobra.Command, args []string) error {
		window := getConfig().General.AlarmWindow
		if alarmWithin != "" {
			d, err := parseWindow(alarmWithin)
			if err != nil {
				return err
			}
			window = d
		}
		if window < 0 {
			return fmt.Errorf("window must be non-negative")
		}

		s, err := store.Load()
		if err != nil {
			return err
		}

		now := time.Now()
		cutoff := now.Add(window)
		var hits []store.Todo
		for _, t := range s.ActiveTodos() {
			if t.DueAt == nil {
				continue
			}
			if t.DueAt.After(cutoff) {
				continue
			}
			if alarmSpace != "" && t.Namespace != alarmSpace {
				continue
			}
			hits = append(hits, t)
		}
		sort.SliceStable(hits, func(i, j int) bool {
			return hits[i].DueAt.Before(*hits[j].DueAt)
		})

		switch alarmFormat {
		case "", "pretty":
			if len(hits) == 0 {
				fmt.Println(c(dim, "No alarms."))
				return nil
			}
			showNS := alarmSpace == ""
			renderList(todoRows(hits, showNS), getConfig().General.LineWidth)
		case "plain":
			for _, t := range hits {
				label, _ := formatDue(*t.DueAt, now)
				ns := ""
				if alarmSpace == "" {
					ns = t.Namespace + " "
				}
				fmt.Printf("[%s] %s%s (%s)\n", t.ID, ns, t.Text, label)
			}
		default:
			return fmt.Errorf("unknown format %q (supported: pretty, plain)", alarmFormat)
		}
		return nil
	},
}

func init() {
	alarmCmd.Flags().StringVarP(&alarmWithin, "within", "w", "", "Window before due date to alarm on (e.g. 30m, 1h, 2d). Defaults to config.")
	alarmCmd.Flags().StringVarP(&alarmFormat, "format", "f", "pretty", "Output format: pretty or plain")
	alarmCmd.Flags().StringVarP(&alarmSpace, "space", "s", "", "Limit to a specific space (default: all spaces)")
}
