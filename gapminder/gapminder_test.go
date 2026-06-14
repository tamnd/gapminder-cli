package gapminder_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tamnd/gapminder-cli/gapminder"
)

const conceptsCSV = `concept,concept_type,name,description
country,entity_domain,Country,Geographic entity
life_expectancy_years,measure,Life Expectancy,Average life expectancy at birth
income_per_person,measure,Income per Person,GDP per capita adjusted for PPP
`

const countriesCSV = `country,name,world_4region
usa,United States,americas
deu,Germany,europe
`

const datapointsCSV = `geo,time,life_expectancy_years
afg,1960,32.5
usa,1960,69.8
deu,1960,69.1
`

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "ddf--concepts.csv"):
			_, _ = w.Write([]byte(conceptsCSV))
		case strings.HasSuffix(r.URL.Path, "ddf--entities--geo--country.csv"):
			_, _ = w.Write([]byte(countriesCSV))
		case strings.Contains(r.URL.Path, "ddf--datapoints--life_expectancy_years--by--geo--time.csv"):
			_, _ = w.Write([]byte(datapointsCSV))
		default:
			http.NotFound(w, r)
		}
	}))
}

func TestIndicators(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	cfg := gapminder.DefaultConfig()
	cfg.BaseURL = srv.URL
	cfg.Rate = 0

	c := gapminder.NewClient(cfg)
	indicators, err := c.Indicators(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	// CSV has 3 rows: 1 entity_domain + 2 measure — expect 2
	if len(indicators) != 2 {
		t.Fatalf("got %d indicators, want 2", len(indicators))
	}
	if indicators[0].ID != "life_expectancy_years" {
		t.Errorf("first indicator ID = %q, want life_expectancy_years", indicators[0].ID)
	}
	if indicators[0].Rank != 1 {
		t.Errorf("rank = %d, want 1", indicators[0].Rank)
	}
	if indicators[1].ID != "income_per_person" {
		t.Errorf("second indicator ID = %q, want income_per_person", indicators[1].ID)
	}
}

func TestCountries(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	cfg := gapminder.DefaultConfig()
	cfg.BaseURL = srv.URL
	cfg.Rate = 0

	c := gapminder.NewClient(cfg)
	countries, err := c.Countries(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(countries) != 2 {
		t.Fatalf("got %d countries, want 2", len(countries))
	}
	if countries[0].Code != "usa" {
		t.Errorf("code = %q, want usa", countries[0].Code)
	}
	if countries[0].Name != "United States" {
		t.Errorf("name = %q, want United States", countries[0].Name)
	}
	if countries[0].Region != "americas" {
		t.Errorf("region = %q, want americas", countries[0].Region)
	}
	if countries[1].Code != "deu" {
		t.Errorf("code = %q, want deu", countries[1].Code)
	}
}

func TestData(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	cfg := gapminder.DefaultConfig()
	cfg.BaseURL = srv.URL
	cfg.Rate = 0

	c := gapminder.NewClient(cfg)
	points, err := c.Data(context.Background(), "life_expectancy_years", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(points) != 3 {
		t.Fatalf("got %d data points, want 3", len(points))
	}
	if points[0].Country != "afg" {
		t.Errorf("country = %q, want afg", points[0].Country)
	}
	if points[0].Year != 1960 {
		t.Errorf("year = %d, want 1960", points[0].Year)
	}
	if points[0].Value != 32.5 {
		t.Errorf("value = %v, want 32.5", points[0].Value)
	}
	if points[0].Indicator != "life_expectancy_years" {
		t.Errorf("indicator = %q, want life_expectancy_years", points[0].Indicator)
	}
	if points[2].Rank != 3 {
		t.Errorf("rank = %d, want 3", points[2].Rank)
	}
}
