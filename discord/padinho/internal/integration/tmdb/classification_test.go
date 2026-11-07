package tmdb

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

// TestMissingClassifications treats missing, null and empty appended data as fetched empty lists.
func TestMissingClassifications(t *testing.T) {
	for _, suffix := range []string{
		"", `,"release_dates":null,"content_ratings":null`,
		`,"release_dates":{},"content_ratings":{}`,
		`,"release_dates":{"results":[]},"content_ratings":{"results":[]}`,
	} {
		for _, kind := range []string{"movie", "tv"} {
			body := `{"id":1,"original_title":"Movie","release_date":"2020-01-01","original_name":"Series","first_air_date":"2020-01-01"` + suffix + "}"
			client, err := New("test", http.Client{Transport: transport(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
			})})
			if err != nil {
				t.Fatal(err)
			}
			title, err := client.Details(kind, 1)
			if err != nil || title.ContentRatings == nil || len(title.ContentRatings) != 0 {
				t.Fatalf("%s/%s empty classifications: %#v %v", kind, suffix, title, err)
			}
		}
	}
}
