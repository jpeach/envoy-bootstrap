package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/jpeach/envoy-bootstrap/pkg/cli"
)

// PROGNAME ...
const PROGNAME = "envoy-bootstrap"

// root represents the base command when called without any subcommands
var root = applyDefaults(&cobra.Command{
	Use:   fmt.Sprintf("%s CMD [FLAGS ...]", PROGNAME),
	Short: "Envoy bootstrapping tool",
})

func applyDefaults(c *cobra.Command) *cobra.Command {
	c.SilenceUsage = true
	c.SilenceErrors = true
	c.DisableFlagsInUseLine = true

	return c
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", PROGNAME, err)
		os.Exit(1)
	}
}

func init() {
	root.AddCommand(applyDefaults(cli.NewRunCommand()))
}
