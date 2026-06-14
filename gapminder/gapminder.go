// Package gapminder is the library behind the gapminder command line:
// the HTTP client, request shaping, and the typed data models for Gapminder
// global development data fetched from the open-numbers GitHub repository.
package gapminder

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const DefaultUserAgent = "Mozilla/5.0 (compatible; gapminder-cli/0.1; +https://github.com/tamnd/gapminder-cli)"

// Config holds constructor parameters.
type Config struct {
	BaseURL   string
	UserAgent string
	Rate      time.Duration
	Retries   int
	Timeout   time.Duration
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		BaseURL:   "https://raw.githubusercontent.com/open-numbers/ddf--gapminder--systema_globalis/master",
		UserAgent: DefaultUserAgent,
		Rate:      500 * time.Millisecond,
		Retries:   3,
		Timeout:   30 * time.Second,
	}
}

// Client talks to the Gapminder open-numbers GitHub repository.
type Client struct {
	cfg        Config
	httpClient *http.Client
	mu         sync.Mutex
	last       time.Time
}

// NewClient returns a Client with the given config.
func NewClient(cfg Config) *Client {
	return &Client{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: cfg.Timeout},
	}
}

func (c *Client) get(ctx context.Context, rawURL string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.cfg.Retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		b, retry, err := c.do(ctx, rawURL)
		if err == nil {
			return b, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, fmt.Errorf("get: %w", lastErr)
}

func (c *Client) do(ctx context.Context, rawURL string) ([]byte, bool, error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.cfg.UserAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("http %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("http %d", resp.StatusCode)
	}

	b, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return nil, true, err
	}
	return b, false, nil
}

func (c *Client) pace() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cfg.Rate <= 0 {
		return
	}
	if wait := c.cfg.Rate - time.Since(c.last); wait > 0 {
		time.Sleep(wait)
	}
	c.last = time.Now()
}

func backoff(attempt int) time.Duration {
	d := time.Duration(attempt) * 500 * time.Millisecond
	if d > 5*time.Second {
		d = 5 * time.Second
	}
	return d
}

// colIndex returns the index of name in header, or -1 if not found.
func colIndex(header []string, name string) int {
	for i, h := range header {
		if h == name {
			return i
		}
	}
	return -1
}

// Indicators fetches all Gapminder indicators (concept_type == "measure").
func (c *Client) Indicators(ctx context.Context, limit int) ([]Indicator, error) {
	url := c.cfg.BaseURL + "/ddf--concepts.csv"
	raw, err := c.get(ctx, url)
	if err != nil {
		return nil, err
	}

	r := csv.NewReader(bytes.NewReader(raw))
	header, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("read concepts header: %w", err)
	}

	conceptCol := colIndex(header, "concept")
	typeCol := colIndex(header, "concept_type")
	nameCol := colIndex(header, "name")
	descCol := colIndex(header, "description")

	if conceptCol < 0 || typeCol < 0 {
		return nil, fmt.Errorf("concepts CSV missing required columns")
	}

	var indicators []Indicator
	rank := 0
	for {
		if limit > 0 && rank >= limit {
			break
		}
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		if row[typeCol] != "measure" {
			continue
		}
		rank++
		ind := Indicator{
			Rank: rank,
			ID:   row[conceptCol],
		}
		if nameCol >= 0 && nameCol < len(row) {
			ind.Name = row[nameCol]
		}
		if descCol >= 0 && descCol < len(row) {
			ind.Description = row[descCol]
		}
		indicators = append(indicators, ind)
	}
	return indicators, nil
}

// Countries fetches all Gapminder countries.
func (c *Client) Countries(ctx context.Context, limit int) ([]Country, error) {
	url := c.cfg.BaseURL + "/ddf--entities--geo--country.csv"
	raw, err := c.get(ctx, url)
	if err != nil {
		return nil, err
	}

	r := csv.NewReader(bytes.NewReader(raw))
	header, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("read countries header: %w", err)
	}

	codeCol := colIndex(header, "country")
	nameCol := colIndex(header, "name")
	regionCol := colIndex(header, "world_4region")

	if codeCol < 0 || nameCol < 0 {
		return nil, fmt.Errorf("countries CSV missing required columns")
	}

	var countries []Country
	rank := 0
	for {
		if limit > 0 && rank >= limit {
			break
		}
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		name := ""
		if nameCol < len(row) {
			name = strings.TrimSpace(row[nameCol])
		}
		if name == "" {
			continue
		}
		rank++
		co := Country{
			Rank: rank,
			Code: row[codeCol],
			Name: name,
		}
		if regionCol >= 0 && regionCol < len(row) {
			co.Region = row[regionCol]
		}
		countries = append(countries, co)
	}
	return countries, nil
}

// Data fetches time-series datapoints for a given indicator.
func (c *Client) Data(ctx context.Context, indicator string, limit int) ([]DataPoint, error) {
	url := fmt.Sprintf(
		"%s/countries-etc-datapoints/ddf--datapoints--%s--by--geo--time.csv",
		c.cfg.BaseURL, indicator,
	)
	raw, err := c.get(ctx, url)
	if err != nil {
		return nil, err
	}

	r := csv.NewReader(bytes.NewReader(raw))
	header, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("read datapoints header: %w", err)
	}

	geoCol := colIndex(header, "geo")
	timeCol := colIndex(header, "time")
	valueCol := colIndex(header, indicator)

	// Fallback: if indicator column not found by name, use column 2
	if geoCol < 0 {
		geoCol = 0
	}
	if timeCol < 0 {
		timeCol = 1
	}
	if valueCol < 0 && len(header) > 2 {
		valueCol = 2
	}
	if valueCol < 0 {
		return nil, fmt.Errorf("datapoints CSV: cannot find value column for %q", indicator)
	}

	var points []DataPoint
	rank := 0
	for {
		if limit > 0 && rank >= limit {
			break
		}
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		if valueCol >= len(row) {
			continue
		}
		valStr := strings.TrimSpace(row[valueCol])
		if valStr == "" {
			continue
		}
		val, err := strconv.ParseFloat(valStr, 64)
		if err != nil {
			continue
		}
		year, err := strconv.Atoi(strings.TrimSpace(row[timeCol]))
		if err != nil {
			continue
		}
		rank++
		points = append(points, DataPoint{
			Rank:      rank,
			Country:   row[geoCol],
			Year:      year,
			Value:     val,
			Indicator: indicator,
		})
	}
	return points, nil
}
