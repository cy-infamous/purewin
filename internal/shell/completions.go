package shell

import (
	"strings"
)

// ─── Completions Component ───────────────────────────────────────────────────
// Dumb component (crush pattern): exposes methods, no Update(). The main
// model calls Open/Close/Filter/Navigate and renders via Render().

// Completions manages the slash-command autocomplete popup.
type Completions struct {
	all      []CmdDef // full command list
	filtered []CmdDef // filtered by current query
	cursor   int      // selected index in filtered list
	open     bool     // whether popup is visible
	query    string   // current filter string (without leading /)
}

// NewCompletions creates a Completions component with the given command list.
func NewCompletions(cmds []CmdDef) *Completions {
	return &Completions{
		all:      cmds,
		filtered: cmds,
	}
}

// Open shows the completions popup and resets the filter.
func (c *Completions) Open() {
	c.open = true
	c.query = ""
	c.cursor = 0
	c.filtered = c.all
}

// Close hides the completions popup.
func (c *Completions) Close() {
	c.open = false
	c.query = ""
	c.cursor = 0
}

// IsOpen returns whether the popup is visible.
func (c *Completions) IsOpen() bool {
	return c.open
}

// Filter updates the filtered list based on the query string.
// The query should NOT include the leading slash.
func (c *Completions) Filter(query string) {
	c.query = strings.ToLower(query)
	c.filtered = make([]CmdDef, 0, len(c.all))
	for _, cmd := range c.all {
		if c.query == "" || strings.Contains(strings.ToLower(cmd.Name), c.query) {
			c.filtered = append(c.filtered, cmd)
		}
	}
	// Clamp cursor.
	if c.cursor >= len(c.filtered) {
		if len(c.filtered) > 0 {
			c.cursor = len(c.filtered) - 1
		} else {
			c.cursor = 0
		}
	}
}

// MoveUp moves the cursor up (wraps around).
func (c *Completions) MoveUp() {
	if len(c.filtered) == 0 {
		return
	}
	if c.cursor > 0 {
		c.cursor--
	} else {
		c.cursor = len(c.filtered) - 1
	}
}

// MoveDown moves the cursor down (wraps around).
func (c *Completions) MoveDown() {
	if len(c.filtered) == 0 {
		return
	}
	if c.cursor < len(c.filtered)-1 {
		c.cursor++
	} else {
		c.cursor = 0
	}
}

// Selected returns the currently highlighted command, or nil if empty.
func (c *Completions) Selected() *CmdDef {
	if len(c.filtered) == 0 {
		return nil
	}
	cmd := c.filtered[c.cursor]
	return &cmd
}

// Filtered returns the current filtered command list.
func (c *Completions) Filtered() []CmdDef {
	return c.filtered
}

// Cursor returns the current cursor position.
func (c *Completions) Cursor() int {
	return c.cursor
}

// builtin max() used from Go 1.21+ — no custom max needed.
