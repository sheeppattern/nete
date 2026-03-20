package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/sheeppattern/zk/internal/store"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize the Zettelkasten storage",
	Long:    "Creates the root directory structure for the Zettelkasten memory store.",
	Example: `  zk init
  zk init --path /custom/store`,
	RunE: func(cmd *cobra.Command, args []string) error {
		storePath := getStorePath(cmd)

		s := store.NewStore(storePath)
		if err := s.Init(); err != nil {
			return fmt.Errorf("failed to initialize store: %w", err)
		}

		statusf("initialized zk store at %s", storePath)

		// Install global agent skill files (non-fatal on failure).
		if err := WriteGlobalAgentFiles(); err != nil {
			fmt.Fprintln(os.Stderr, "warning: failed to write agent skill files:", err)
		}

		return nil
	},
}

func init() {
	initCmd.Flags().String("path", "", "path for the zk store (overrides default)")
}
