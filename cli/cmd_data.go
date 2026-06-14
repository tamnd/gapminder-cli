package cli

import (
	"github.com/spf13/cobra"
)

func (a *App) dataCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "data <indicator>",
		Short: "Show time-series data for a Gapminder indicator",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			limit := a.effectiveLimit(0)
			data, err := a.client.Data(cmd.Context(), args[0], limit)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(data, len(data))
		},
	}
	return cmd
}
