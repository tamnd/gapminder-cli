package cli

import (
	"github.com/spf13/cobra"
)

func (a *App) countriesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "countries",
		Short: "List all Gapminder countries with code, name, and region",
		RunE: func(cmd *cobra.Command, _ []string) error {
			limit := a.effectiveLimit(0)
			countries, err := a.client.Countries(cmd.Context(), limit)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(countries, len(countries))
		},
	}
}
