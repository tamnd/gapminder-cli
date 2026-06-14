package cli

import (
	"github.com/spf13/cobra"
)

func (a *App) indicatorsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "indicators",
		Short: "List all Gapminder data indicators",
		RunE: func(cmd *cobra.Command, _ []string) error {
			limit := a.effectiveLimit(0)
			indicators, err := a.client.Indicators(cmd.Context(), limit)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(indicators, len(indicators))
		},
	}
}
