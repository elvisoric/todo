package cmd

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

type listRow struct {
	ID      string // pre-formatted (color/brackets)
	NS      string // pre-formatted; empty string means "don't render this column"
	Text    string // raw, wrapped at render time
	Due     string // pre-formatted
	Created string // pre-formatted
	Done    string // pre-formatted, optional trailing suffix
}

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)

func visibleLen(s string) int {
	return utf8.RuneCountInString(ansiRe.ReplaceAllString(s, ""))
}

func padVis(s string, width int) string {
	diff := width - visibleLen(s)
	if diff <= 0 {
		return s
	}
	return s + strings.Repeat(" ", diff)
}

// wrapText word-wraps s to at most width runes per line. Long single words
// are hard-split. Empty input returns a single empty line.
func wrapText(s string, width int) []string {
	if width <= 0 {
		return []string{s}
	}
	if utf8.RuneCountInString(s) <= width {
		return []string{s}
	}
	var lines []string
	var cur strings.Builder
	curLen := 0
	flush := func() {
		if cur.Len() > 0 {
			lines = append(lines, cur.String())
			cur.Reset()
			curLen = 0
		}
	}
	for _, word := range strings.Fields(s) {
		wLen := utf8.RuneCountInString(word)
		if wLen > width {
			flush()
			// hard-split oversized words
			runes := []rune(word)
			for len(runes) > width {
				lines = append(lines, string(runes[:width]))
				runes = runes[width:]
			}
			cur.WriteString(string(runes))
			curLen = len(runes)
			continue
		}
		if curLen == 0 {
			cur.WriteString(word)
			curLen = wLen
		} else if curLen+1+wLen <= width {
			cur.WriteByte(' ')
			cur.WriteString(word)
			curLen += 1 + wLen
		} else {
			flush()
			cur.WriteString(word)
			curLen = wLen
		}
	}
	flush()
	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}

// renderList prints rows as a padded, word-wrapped table. The text column
// absorbs any slack so the overall line stays within lineWidth runes.
func renderList(rows []listRow, lineWidth int) {
	if len(rows) == 0 {
		return
	}
	const (
		leading = 2
		gap     = 2
		minText = 20
	)

	idW, nsW, dueW, createdW, doneW := 0, 0, 0, 0, 0
	hasNS := false
	for _, r := range rows {
		if v := visibleLen(r.ID); v > idW {
			idW = v
		}
		if r.NS != "" {
			hasNS = true
		}
		if v := visibleLen(r.NS); v > nsW {
			nsW = v
		}
		if v := visibleLen(r.Due); v > dueW {
			dueW = v
		}
		if v := visibleLen(r.Created); v > createdW {
			createdW = v
		}
		if v := visibleLen(r.Done); v > doneW {
			doneW = v
		}
	}

	// Width of everything except the text column itself.
	fixedW := leading + idW + gap
	if hasNS {
		fixedW += nsW + gap
	}
	fixedW += gap + dueW + gap + createdW
	if doneW > 0 {
		fixedW += 1 + doneW
	}

	textW := lineWidth - fixedW
	if textW < minText {
		textW = minText
	}

	contPad := strings.Repeat(" ", leading+idW+gap)
	if hasNS {
		contPad += strings.Repeat(" ", nsW+gap)
	}

	for _, r := range rows {
		lines := wrapText(r.Text, textW)

		var b strings.Builder
		b.WriteString(strings.Repeat(" ", leading))
		b.WriteString(padVis(r.ID, idW))
		b.WriteString(strings.Repeat(" ", gap))
		if hasNS {
			b.WriteString(padVis(r.NS, nsW))
			b.WriteString(strings.Repeat(" ", gap))
		}
		b.WriteString(padVis(lines[0], textW))
		b.WriteString(strings.Repeat(" ", gap))
		b.WriteString(padVis(r.Due, dueW))
		b.WriteString(strings.Repeat(" ", gap))
		b.WriteString(padVis(r.Created, createdW))
		if r.Done != "" {
			b.WriteString(" ")
			b.WriteString(r.Done)
		}
		fmt.Println(b.String())

		for i := 1; i < len(lines); i++ {
			fmt.Println(contPad + lines[i])
		}
	}
}
