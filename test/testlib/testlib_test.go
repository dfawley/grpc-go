package testlib

import (
	"strings"
	"testing"
)

func TestReplaceInt(t *testing.T) {
	const oldI = 123
	const newI = 321
	i := oldI
	restore := Replace(&i, newI)
	if i != newI {
		t.Errorf("Error setting value: %v vs. %v", i, newI)
	}
	restore()
	if i != oldI {
		t.Errorf("Error restoring value: %v vs. %v", i, oldI)
	}
}

func TestReplaceFunc(t *testing.T) {
	const oldI = 123
	const newI = 321
	oldF := func() int { return oldI }
	newF := func() int { return newI }
	restore := Replace(&oldF, newF)
	if oldF() != newI {
		t.Errorf("Error setting value: %v vs. %v", oldF(), newI)
	}
	restore()
	if oldF() != oldI {
		t.Errorf("Error restoring value: %v vs. %v", oldF(), oldI)
	}
}

func TestReplaceTypeError(t *testing.T) {
	defer func() {
		const panicMsg = "type string is not assignable to type int"
		if r := recover(); r == nil || !strings.Contains(r.(string), panicMsg) {
			t.Errorf("Did not receive expected panic: %v vs. %v", r, panicMsg)
		}
	}()
	i := 123
	Replace(&i, "")
}
