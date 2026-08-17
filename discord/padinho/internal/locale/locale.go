// Package locale contains helpers shared by the application's localized copy.
package locale

import "unicode"

// Capitalize uppercases the first Unicode letter in value.
func Capitalize(value string) string {
	for index, character := range value {
		return string(unicode.ToUpper(character)) + value[index+len(string(character)):]
	}
	return value
}
