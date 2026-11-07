// Package tmdb isolates the third-party TMDB wrapper and its transport policy.
package tmdb

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	wrapper "github.com/cyruzin/golang-tmdb"
	app "github.com/kiLeo13/SaladaFun/discord/padinho/internal/application/letterboxd"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/config"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
)

// Client fetches complete Portuguese movie/series metadata with bearer auth.
type Client struct{ client *wrapper.Client }

// ConfigReader supplies the existing database-backed configuration contract.
type ConfigReader interface{ Get(string) (string, error) }

// FromConfig loads the required token without exposing its value on failures.
func FromConfig(source ConfigReader, client http.Client) (*Client, error) {
	token, err := source.Get(config.TMDBToken)
	if err != nil {
		return nil, errors.New("tmdb.token configuration unavailable")
	}
	return New(token, client)
}

// New configures a shared client; retries are disabled to bound latency.
func New(token string, client http.Client) (*Client, error) {
	if strings.TrimSpace(token) == "" {
		return nil, errors.New("tmdb.token is empty")
	}
	c, err := wrapper.InitV4(token)
	if err != nil {
		return nil, err
	}
	if client.Timeout <= 0 {
		client.Timeout = 5 * time.Second
	}
	c.SetClientConfig(client)
	return &Client{c}, nil
}

// Details retrieves and validates a complete snapshot without exposing tokens.
func (c *Client) Details(kind string, id uint64) (*entity.LetterboxdTitle, error) {
	if !app.ValidIdentity(kind, id) || id > uint64(^uint(0)>>1) {
		return nil, app.ErrInvalid
	}
	t := &entity.LetterboxdTitle{MediaType: kind, TMDBID: id}
	options := map[string]string{"language": "pt-BR"}
	var date string
	if kind == "movie" {
		m, err := c.client.GetMovieDetails(int(id), options)
		if err != nil {
			return nil, errors.New("TMDB movie details unavailable")
		}
		if m.ID != int64(id) {
			return nil, app.ErrInvalid
		}
		t.OriginalTitle = m.OriginalTitle
		t.LocalizedTitle = m.Title
		t.Synopsis = m.Overview
		t.PosterPath = m.PosterPath
		t.BackdropPath = m.BackdropPath
		date = m.ReleaseDate
		if t.BackdropPath == "" {
			t.BackdropPath = m.BelongsToCollection.BackdropPath
		}
	} else {
		m, err := c.client.GetTVDetails(int(id), options)
		if err != nil {
			return nil, errors.New("TMDB series details unavailable")
		}
		if m.ID != int64(id) {
			return nil, app.ErrInvalid
		}
		t.OriginalTitle = m.OriginalName
		t.LocalizedTitle = m.Name
		t.Synopsis = m.Overview
		t.PosterPath = m.PosterPath
		t.BackdropPath = m.BackdropPath
		date = m.FirstAirDate
	}
	if t.OriginalTitle == "" {
		t.OriginalTitle = t.LocalizedTitle
	}
	if len(date) == 4 {
		t.ReleaseYear, _ = strconv.Atoi(date)
	} else if d, err := time.Parse("2006-01-02", date); err == nil {
		t.ReleaseDate = &d
		t.ReleaseYear = d.Year()
	}
	if err := app.ValidateTitle(t); err != nil {
		return nil, err
	}
	return t, nil
}

// Search maps remote multi-search suggestions without writing cache records.
func (c *Client) Search(query string, page int) ([]entity.LetterboxdTitle, error) {
	if strings.TrimSpace(query) == "" {
		return nil, nil
	}
	result, err := c.client.GetSearchMulti(query, map[string]string{"language": "pt-BR", "page": strconv.Itoa(page), "include_adult": "false"})
	if err != nil {
		return nil, errors.New("TMDB search unavailable")
	}
	if result.SearchMultiResults == nil {
		return nil, nil
	}
	var titles []entity.LetterboxdTitle
	for _, m := range result.Results {
		if !app.ValidIdentity(m.MediaType, uint64(m.ID)) || m.ID <= 0 {
			continue
		}
		t := entity.LetterboxdTitle{MediaType: m.MediaType, TMDBID: uint64(m.ID), OriginalTitle: m.OriginalTitle, LocalizedTitle: m.Title}
		date := m.ReleaseDate
		if m.MediaType == "tv" {
			t.OriginalTitle = m.OriginalName
			t.LocalizedTitle = m.Name
			date = m.FirstAirDate
		}
		if t.OriginalTitle == "" {
			t.OriginalTitle = t.LocalizedTitle
		}
		if len(date) >= 4 {
			t.ReleaseYear, _ = strconv.Atoi(date[:4])
		}
		titles = append(titles, t)
	}
	return titles, nil
}
