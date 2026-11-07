package main

import (
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func testModel() model {
	m := newModel()
	m.tool = Tool{Name: "t", Flags: []Flag{
		{Name: "text", Kind: KindString},
		{Name: "on", Kind: KindBool},
		{Name: "choice", Kind: KindChoice, Value: "a", Choices: []string{"a", "b"}},
	}}
	m.screen = screenConfig
	m.syncInput()
	return m
}

func press(t *testing.T, m model, keys ...string) model {
	t.Helper()
	for _, k := range keys {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
		switch k {
		case "esc":
			msg = tea.KeyMsg{Type: tea.KeyEsc}
		case "enter":
			msg = tea.KeyMsg{Type: tea.KeyEnter}
		case " ":
			msg = tea.KeyMsg{Type: tea.KeySpace, Runes: []rune(" ")}
		}
		next, cmd := m.Update(msg)
		m = next.(model)
		// Deliver filter results as the runtime would.
		if m.screen == screenList {
			m.list = settleList(m.list, cmd)
		}
	}
	return m
}

func TestJKLeavesInsertWithoutTypingJ(t *testing.T) {
	m := press(t, testModel(), "i", "a", "b", "j", "k")
	if m.insert {
		t.Fatal("jk should leave insert mode")
	}
	if got := m.tool.Flags[0].Value; got != "ab" {
		t.Fatalf("value = %q, want %q", got, "ab")
	}
	if got := m.input.Position(); got != 1 {
		t.Fatalf("cursor = %d, want 1 (stepped back onto the last char)", got)
	}
}

func TestSlowJKIsLiteral(t *testing.T) {
	m := press(t, testModel(), "i", "j")
	m.lastJ = time.Now().Add(-2 * jkTimeout)
	m = press(t, m, "k")
	if !m.insert || m.tool.Flags[0].Value != "jk" {
		t.Fatalf("insert=%v value=%q, want literal jk", m.insert, m.tool.Flags[0].Value)
	}
}

func TestJInterruptedIsLiteral(t *testing.T) {
	m := press(t, testModel(), "i", "j", "x", "k")
	if !m.insert || m.tool.Flags[0].Value != "jxk" {
		t.Fatalf("insert=%v value=%q", m.insert, m.tool.Flags[0].Value)
	}
}

func TestNormalKeysDoNotType(t *testing.T) {
	m := press(t, testModel(), "z", "p")
	if m.tool.Flags[0].Value != "" {
		t.Fatalf("value = %q, normal mode must not type", m.tool.Flags[0].Value)
	}
}

func TestNavigationAndFlagEditing(t *testing.T) {
	m := press(t, testModel(), "j")
	if m.cursor != 1 {
		t.Fatalf("j: cursor = %d", m.cursor)
	}
	if m = press(t, m, " "); !m.tool.Flags[1].Enabled {
		t.Fatal("space should toggle the bool")
	}
	m = press(t, m, "G", "l")
	if m.cursor != 2 || m.tool.Flags[2].Value != "b" {
		t.Fatalf("G l: cursor=%d value=%q", m.cursor, m.tool.Flags[2].Value)
	}
	if m = press(t, m, "g", "g"); m.cursor != 0 {
		t.Fatalf("gg: cursor = %d", m.cursor)
	}
	if m = press(t, m, "g", "j"); m.cursor != 0 || m.pending != "" {
		t.Fatalf("g j should cancel: cursor=%d pending=%q", m.cursor, m.pending)
	}
}

func TestNormalEdits(t *testing.T) {
	type tc struct {
		keys []string
		want string
		ins  bool
	}
	for name, c := range map[string]tc{
		"x at end":    {[]string{"x"}, "hell", false},
		"0 x":         {[]string{"0", "x"}, "ello", false},
		"h D":         {[]string{"h", "D"}, "hel", false},
		"dd":          {[]string{"d", "d"}, "", false},
		"cc":          {[]string{"c", "c", "y"}, "y", true},
		"C":           {[]string{"h", "C", "y"}, "hely", true},
		"A":           {[]string{"A", "!"}, "hello!", true},
		"I":           {[]string{"I", "!"}, "!hello", true},
		"a":           {[]string{"0", "a", "!"}, "h!ello", true},
		"d then x":    {[]string{"d", "x"}, "hello", false},
		"esc returns": {[]string{"i", "esc", "x"}, "helo", false},
	} {
		m := press(t, testModel(), "i", "h", "e", "l", "l", "o", "esc")
		m = press(t, m, c.keys...)
		if got := m.tool.Flags[0].Value; got != c.want || m.insert != c.ins {
			t.Errorf("%s: value=%q insert=%v, want %q %v", name, got, m.insert, c.want, c.ins)
		}
		if got := m.input.Value(); got != c.want {
			t.Errorf("%s: input=%q out of sync with flag %q", name, got, c.want)
		}
	}
}

func TestEscBackToListOnlyFromNormal(t *testing.T) {
	m := press(t, testModel(), "i", "esc")
	if m.screen != screenConfig {
		t.Fatal("first esc should only leave insert mode")
	}
	if m = press(t, m, "esc"); m.screen != screenList {
		t.Fatal("esc in normal mode should go back to the list")
	}
}

func TestCatalogSwitch(t *testing.T) {
	m := newModel()
	start := m.modeIdx
	if m = press(t, m, "L"); m.modeIdx == start {
		t.Fatal("L should switch catalog")
	}
	if m = press(t, m, "H"); m.modeIdx != start {
		t.Fatal("H should switch back")
	}
}

func TestJKInFilterKeepsQuery(t *testing.T) {
	m := newModel()
	m.list.SetSize(80, 40)
	name := m.list.Items()[0].(Tool).Name
	keys := []string{"/"}
	for _, r := range name {
		keys = append(keys, string(r))
	}
	m = press(t, m, append(keys, "j", "k")...)
	if m.list.FilterState() != list.FilterApplied {
		t.Fatalf("state = %v, want applied", m.list.FilterState())
	}
	if got := m.list.FilterValue(); got != name {
		t.Fatalf("filter = %q, want %q", got, name)
	}
	if len(m.list.VisibleItems()) == 0 {
		t.Fatal("filter should still have matches")
	}
}
