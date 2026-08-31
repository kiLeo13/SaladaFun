package messagecommand

import (
	"strconv"

	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/command"
)

// ExtractSnowflake returns the one positive decimal identifier embedded in an
// entity argument, accepting raw IDs and Discord mention markup alike.
func ExtractSnowflake(argument string) (command.Snowflake, bool) {
	start, end := -1, -1
	for index, runeValue := range argument {
		if runeValue >= '0' && runeValue <= '9' {
			if start == -1 {
				start = index
			}
			end = index + len(string(runeValue))
			continue
		}
		if start != -1 {
			break
		}
	}
	if start == -1 {
		return "", false
	}
	value, err := strconv.ParseUint(argument[start:end], 10, 64)
	if err != nil || value == 0 {
		return "", false
	}
	return command.Snowflake(strconv.FormatUint(value, 10)), true
}
