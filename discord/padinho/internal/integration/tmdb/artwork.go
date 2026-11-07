package tmdb

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	_ "image/jpeg" // Register TMDB JPEG artwork.
	_ "image/png"  // Register PNG artwork.
	"io"
	"net/http"
	"strings"
	"time"
)

const maxArtworkBytes = 2 << 20
const maxArtworkPixels = 2000000

// Artwork extracts accents from bounded CDN downloads.
type Artwork struct{ client http.Client }

// NewArtwork configures the immutable downloader.
func NewArtwork(client http.Client) *Artwork {
	if client.Timeout <= 0 {
		client.Timeout = 3 * time.Second
	}
	return &Artwork{client}
}

// Accent confines downloads to TMDB and checks dimensions before decoding.
func (a *Artwork) Accent(ctx context.Context, path string) (uint32, error) {
	if !strings.HasPrefix(path, "/") || strings.ContainsAny(path, "?#\\") || strings.Contains(path, "..") || strings.HasPrefix(path, "//") {
		return 0, errors.New("invalid artwork path")
	}
	req, err := http.NewRequestWithContext(ctx, "GET", "https://image.tmdb.org/t/p/w300"+path, nil)
	if err != nil {
		return 0, err
	}
	client := a.client
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	res, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return 0, errors.New("artwork unavailable")
	}
	data, err := io.ReadAll(io.LimitReader(res.Body, maxArtworkBytes+1))
	if err != nil {
		return 0, err
	}
	if len(data) > maxArtworkBytes {
		return 0, errors.New("artwork too large")
	}
	config, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return 0, err
	}
	if config.Width <= 0 || config.Height <= 0 || int64(config.Width)*int64(config.Height) > maxArtworkPixels {
		return 0, errors.New("artwork dimensions too large")
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return 0, err
	}
	return dominant(img), nil
}

// dominant uses a quantized center-weighted histogram with stable ties.
func dominant(img image.Image) uint32 {
	type bucket struct{ weight, r, g, b uint64 }
	var buckets [4096]bucket
	bounds := img.Bounds()
	for y := 0; y < 36; y++ {
		for x := 0; x < 64; x++ {
			c := color.NRGBAModel.Convert(img.At(bounds.Min.X+x*bounds.Dx()/64, bounds.Min.Y+y*bounds.Dy()/36)).(color.NRGBA)
			if c.A == 0 {
				continue
			}
			weight := uint64(1)
			if x >= 16 && x < 48 && y >= 9 && y < 27 {
				weight = 3
			}
			if (c.R < 25 && c.G < 25 && c.B < 25) || (c.R > 230 && c.G > 230 && c.B > 230) {
				weight = 1
			} else {
				weight *= 4
			}
			weight *= uint64(c.A)
			key := int(c.R>>4)<<8 | int(c.G>>4)<<4 | int(c.B>>4)
			b := &buckets[key]
			b.weight += weight
			b.r += uint64(c.R) * weight
			b.g += uint64(c.G) * weight
			b.b += uint64(c.B) * weight
		}
	}
	best := bucket{}
	for _, b := range buckets {
		if b.weight > best.weight {
			best = b
		}
	}
	if best.weight == 0 {
		return 0x5865F2
	}
	r, g, b := best.r/best.weight, best.g/best.weight, best.b/best.weight
	if r+g+b < 90 {
		r += 30
		g += 30
		b += 30
	}
	if r+g+b > 690 {
		r = r * 3 / 4
		g = g * 3 / 4
		b = b * 3 / 4
	}
	return uint32(r<<16 | g<<8 | b)
}
