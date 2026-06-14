package gapminder

// Indicator is a Gapminder data indicator (e.g. life_expectancy_years).
type Indicator struct {
	Rank        int    `json:"rank"`
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Country is a geographic entity tracked by Gapminder.
type Country struct {
	Rank   int    `json:"rank"`
	Code   string `json:"code"`
	Name   string `json:"name"`
	Region string `json:"region"`
}

// DataPoint is a single observation: a country, year, and indicator value.
type DataPoint struct {
	Rank      int     `json:"rank"`
	Country   string  `json:"country"`
	Year      int     `json:"year"`
	Value     float64 `json:"value"`
	Indicator string  `json:"indicator"`
}
