package cmd

import (
	"github.com/spf13/cobra"
)

var (
	// Version information - will be set at build time
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long: `Print detailed version information including build details.

Environment Variables:
  See 'replbac --help' for full environment variable documentation.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Printf("replbac version %s\n", Version)
		cmd.Printf("Git commit: %s\n", GitCommit)
		cmd.Printf("Built: %s\n", BuildDate)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
