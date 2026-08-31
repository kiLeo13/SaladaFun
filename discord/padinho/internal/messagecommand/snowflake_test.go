package messagecommand

import "testing"

func TestExtractSnowflakeAcceptsRawIDsAndMentionMarkup(t *testing.T) {
	for input, want := range map[string]string{
		"123456789012345678":     "123456789012345678",
		"<@123456789012345678>":  "123456789012345678",
		"<@!123456789012345678>": "123456789012345678",
		"<#123456789012345678>":  "123456789012345678",
		"<@&123456789012345678>": "123456789012345678",
	} {
		if got, valid := ExtractSnowflake(input); !valid || string(got) != want {
			t.Fatalf("ExtractSnowflake(%q) = %q, %t", input, got, valid)
		}
	}
}

func TestExtractSnowflakeRejectsMissingAndInvalidIDs(t *testing.T) {
	for _, input := range []string{"", "user", "0", "١٢٣", "18446744073709551616"} {
		if got, valid := ExtractSnowflake(input); valid || got != "" {
			t.Fatalf("ExtractSnowflake(%q) = %q, %t", input, got, valid)
		}
	}
}
