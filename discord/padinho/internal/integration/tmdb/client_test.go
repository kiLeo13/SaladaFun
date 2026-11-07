package tmdb

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"strings"
	"testing"
)

type transport func(*http.Request) (*http.Response, error)

// RoundTrip makes provider tests independent of external services.
func (f transport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// TestDetails checks authentication, localization, media mapping, and validation.
func TestDetails(t *testing.T) {
	if _, err := New("", http.Client{}); err == nil {
		t.Fatal("empty token")
	}
	for _, kind := range []string{"movie", "tv"} {
		t.Run(kind, func(t *testing.T) {
			c, err := New("secret", http.Client{Transport: transport(func(r *http.Request) (*http.Response, error) {
				if r.Header.Get("Authorization") != "Bearer secret" || r.URL.Query().Get("language") != "pt-BR" {
					t.Fatal("auth/options")
				}
				if kind == "movie" && r.URL.Query().Get("append_to_response") != "credits,release_dates" {
					t.Fatal("movie credits missing")
				}
				if kind == "tv" && r.URL.Query().Get("append_to_response") != "content_ratings" {
					t.Fatal("TV classifications missing")
				}
				body := `{"id":1,"original_title":"Original","title":"Traduzido","release_date":"2020-02-03","original_name":"Series","name":"Serie","first_air_date":"2021","homepage":"https://example.test/title","genres":[{"name":" Drama "},{"name":"Drama"},{"name":""},{"name":"Comedy, Drama"}],"created_by":[{"name":" Ross Duffer "},{"name":"Matt Duffer"},{"name":"Ross Duffer"}],"number_of_seasons":5,"credits":{"crew":[{"name":" Director, Jr. ","job":"Director"},{"name":"Director, Jr.","job":"Director"},{"name":"Writer","job":"Writer"}]},"release_dates":{"results":[{"iso_3166_1":"BR","release_dates":[{"certification":"16","type":3}]}]},"content_ratings":{"results":[{"iso_3166_1":"BR","rating":"16"}]},"belongs_to_collection":{"backdrop_path":"/collection.jpg"}}`
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
			})})
			if err != nil {
				t.Fatal(err)
			}
			title, err := c.Details(kind, 1)
			if err != nil || title.ReleaseYear < 2020 {
				t.Fatalf("details: %#v %v", title, err)
			}
			if len(title.ContentRatings) != 1 || title.ContentRatings[0].Country != "BR" || title.ContentRatings[0].Certification != "16" {
				t.Fatalf("classification mapping: %#v", title.ContentRatings)
			}
			if kind == "movie" && title.ContentRatings[0].ReleaseType != 3 {
				t.Fatal("release type lost")
			}
			if kind == "tv" && title.ContentRatings[0].ReleaseType != 0 {
				t.Fatal("TV release type")
			}
			if title.Homepage != "https://example.test/title" || len(title.Genres) != 2 || title.Genres[1] != "Comedy, Drama" {
				t.Fatalf("metadata %#v", title)
			}
			if kind == "tv" && (len(title.Creators) != 2 || title.Creators[0] != "Ross Duffer" || title.SeasonCount != 5) {
				t.Fatal("TV creators/seasons")
			}
			if kind == "movie" && (len(title.Creators) != 1 || title.Creators[0] != "Director, Jr." || title.SeasonCount != 0) {
				t.Fatal("movie directors")
			}
			if kind == "movie" && title.BackdropPath != "/collection.jpg" {
				t.Fatal("collection")
			}
			if kind == "tv" && title.ReleaseDate != nil {
				t.Fatal("invented date")
			}
		})
	}
	for _, body := range []string{`{}`, `broken`, `{"id":1,"original_title":"Name","release_date":""}`} {
		c, _ := New("secret", http.Client{Transport: transport(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
		})})
		if _, err := c.Details("movie", 1); err == nil {
			t.Fatal("invalid accepted")
		}
	}
	c, _ := New("secret", http.Client{Transport: transport(func(*http.Request) (*http.Response, error) { return nil, fmt.Errorf("network secret") })})
	if _, err := c.Details("movie", 1); err == nil || strings.Contains(err.Error(), "secret") {
		t.Fatal("leaked error")
	}
	if _, err := c.Details("person", 1); err == nil {
		t.Fatal("identity")
	}
}

// TestArtwork checks decoding, dominant color, path validation and HTTP failures.
func TestArtwork(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			img.Set(x, y, color.NRGBA{R: 200, G: 40, B: 50, A: 255})
		}
	}
	var encoded bytes.Buffer
	png.Encode(&encoded, img)
	a := NewArtwork(http.Client{Transport: transport(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host != "image.tmdb.org" {
			t.Fatal("host")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(encoded.Bytes()))}, nil
	})})
	value, err := a.Accent(context.Background(), "/a.png")
	if err != nil || value != 0xc82832 {
		t.Fatalf("accent %x %v", value, err)
	}
	for _, path := range []string{"https://bad", "//bad", "/../bad", "/bad?query"} {
		if _, err := a.Accent(context.Background(), path); err == nil {
			t.Fatal("path accepted")
		}
	}
	for _, status := range []int{404, 429} {
		a := NewArtwork(http.Client{Transport: transport(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader("bad"))}, nil
		})})
		if _, err := a.Accent(context.Background(), "/a"); err == nil {
			t.Fatal("status")
		}
	}
	if dominant(image.NewNRGBA(image.Rect(0, 0, 1, 1))) != 0x5865F2 {
		t.Fatal("transparent")
	}
}

type configReader struct {
	token string
	err   error
}

// Get verifies the database config key without an external database.
func (c configReader) Get(key string) (string, error) {
	if key != "tmdb.token" {
		panic("wrong config key")
	}
	return c.token, c.err
}

// TestConfiguredClient covers missing, empty and populated database settings.
func TestConfiguredClient(t *testing.T) {
	if _, err := FromConfig(configReader{err: fmt.Errorf("secret")}, http.Client{}); err == nil || strings.Contains(err.Error(), "secret") {
		t.Fatal("config error")
	}
	if _, err := FromConfig(configReader{}, http.Client{}); err == nil {
		t.Fatal("empty config")
	}
	if _, err := FromConfig(configReader{token: "token"}, http.Client{}); err != nil {
		t.Fatal(err)
	}
}

// TestRateLimit verifies bounded failure rather than sleep-based automatic retry.
func TestRateLimit(t *testing.T) {
	calls := 0
	c, _ := New("secret", http.Client{Transport: transport(func(*http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 429, Header: http.Header{"Retry-After": []string{"60"}}, Body: io.NopCloser(strings.NewReader(`{"status_code":25,"status_message":"rate limit"}`))}, nil
	})})
	if _, err := c.Details("tv", 1); err == nil || calls != 1 {
		t.Fatal("rate limit policy")
	}
}

// TestArtworkFailures verifies malformed and oversized payloads never produce accents.
func TestArtworkFailures(t *testing.T) {
	for _, data := range [][]byte{[]byte("bad image"), bytes.Repeat([]byte("x"), maxArtworkBytes+1)} {
		a := NewArtwork(http.Client{Transport: transport(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(data))}, nil
		})})
		if _, err := a.Accent(context.Background(), "/a.png"); err == nil {
			t.Fatal("invalid image")
		}
	}
	a := NewArtwork(http.Client{Transport: transport(func(*http.Request) (*http.Response, error) { return nil, fmt.Errorf("offline") })})
	if _, err := a.Accent(context.Background(), "/a.png"); err == nil {
		t.Fatal("network failure")
	}
	for _, c := range []color.NRGBA{{A: 255}, {R: 255, G: 255, B: 255, A: 255}} {
		img := image.NewNRGBA(image.Rect(0, 0, 1, 1))
		img.Set(0, 0, c)
		if dominant(img) == 0 {
			t.Fatal("invisible accent")
		}
	}
}

// TestSearchMapping verifies typed suggestions and provider-error behavior.
func TestSearchMapping(t *testing.T) {
	c, _ := New("secret", http.Client{Transport: transport(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/3/search/multi" || r.URL.Query().Get("page") != "2" {
			t.Fatal("search request")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"results":[{"id":1,"media_type":"movie","original_title":"Movie","title":"Filme","release_date":"2020-01-02"},{"id":2,"media_type":"tv","original_name":"Series","name":"Serie","first_air_date":"2021-01-01"},{"id":3,"media_type":"person","name":"Person"}]}`))}, nil
	})})
	titles, err := c.Search("name", 2)
	if err != nil || len(titles) != 2 || titles[1].MediaType != "tv" || titles[1].OriginalTitle != "Series" {
		t.Fatalf("mapping %#v %v", titles, err)
	}
	if titles, err := c.Search(" ", 1); err != nil || len(titles) != 0 {
		t.Fatal("blank search")
	}
}
