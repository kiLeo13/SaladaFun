package locale

import "testing"

func TestCapitalize(t *testing.T) {
	for value, want := range map[string]string{
		"january": "January",
		"ábril":   "Ábril",
		"":        "",
	} {
		if got := Capitalize(value); got != want {
			t.Fatalf("Capitalize(%q) = %q, want %q", value, got, want)
		}
	}
}
