package cmd

import (
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Fetch recent activity and render the markdown file (combined fetch + render)",
	Long: `Convenience command that runs ` + "`fetch`" + ` followed by ` + "`render`" + ` in one
invocation. Intended for unattended/scheduled use after the one-time ` + "`auth`" + `
flow has been completed.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := fetchCmd.RunE(cmd, args); err != nil {
			return err
		}
		return renderCmd.RunE(cmd, args)
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}
