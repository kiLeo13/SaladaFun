package command

import (
	"errors"
	"reflect"
	"testing"
)

func TestCommandPathAndOptionValues(t *testing.T) {
	t.Parallel()

	path := CommandPath{Command: "groups", Group: "members", Subcommand: "add"}
	if want := []string{"groups", "members", "add"}; !reflect.DeepEqual(path.Segments(), want) {
		t.Fatalf("Segments() = %v, want %v", path.Segments(), want)
	}
	if (CommandPath{}).key() != "" {
		t.Fatal("empty path has non-empty key")
	}

	source := map[string]any{"text": "hello", "count": int64(2), "flag": true, "user": Snowflake("123")}
	values := NewOptionValues(source)
	source["text"] = "changed"
	if value, err := values.String("text"); err != nil || value != "hello" {
		t.Fatalf("String() = %q, %v", value, err)
	}
	if value, err := values.Integer("count"); err != nil || value != 2 {
		t.Fatalf("Integer() = %d, %v", value, err)
	}
	if value, err := values.Boolean("flag"); err != nil || !value {
		t.Fatalf("Boolean() = %v, %v", value, err)
	}
	if value, err := values.Snowflake("user"); err != nil || value != "123" {
		t.Fatalf("Snowflake() = %q, %v", value, err)
	}
	if _, err := values.String("missing"); !errors.Is(err, ErrOptionMissing) {
		t.Fatalf("missing error = %v", err)
	}
	if _, err := values.String("count"); !errors.Is(err, ErrOptionType) {
		t.Fatalf("type error = %v", err)
	}
	for name, call := range map[string]func() error{
		"integer missing":   func() error { _, err := values.Integer("missing"); return err },
		"integer type":      func() error { _, err := values.Integer("text"); return err },
		"boolean missing":   func() error { _, err := values.Boolean("missing"); return err },
		"boolean type":      func() error { _, err := values.Boolean("text"); return err },
		"snowflake missing": func() error { _, err := values.Snowflake("missing"); return err },
		"snowflake type":    func() error { _, err := values.Snowflake("text"); return err },
	} {
		if err := call(); err == nil {
			t.Fatalf("%s returned no error", name)
		}
	}
}

func TestTypedOptionBuilders(t *testing.T) {
	t.Parallel()

	options := []Option{
		StringOption("text", "Text").Required().Autocomplete(),
		IntegerOption("count", "Count").Required().Autocomplete(),
		BooleanOption("quiet", "Quiet").Required(),
		UserOption("user", "User").Required(),
		ChannelOption("channel", "Channel").Required(),
	}
	wantTypes := []OptionType{OptionTypeString, OptionTypeInteger, OptionTypeBoolean, OptionTypeUser, OptionTypeChannel}
	for index, option := range options {
		definition := option.snapshot()
		if definition.Type != wantTypes[index] || !definition.Required {
			t.Fatalf("option %d = %#v", index, definition)
		}
	}
}
