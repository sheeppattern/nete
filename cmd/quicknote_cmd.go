package cmd

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/spf13/cobra"
	"github.com/sheeppattern/zk/internal/model"
	"github.com/sheeppattern/zk/internal/store"
)

var quicknoteCmd = &cobra.Command{
	Use:   "quicknote <text>",
	Short: "Create a note with minimal input",
	Long:  "Shortcut for note creation: title is auto-derived from the text (truncated at 50 chars), layer defaults to concrete, no tags required.",
	Example: `  zk quicknote "Redis cache hit rate is 95%"
  zk quicknote "This pattern keeps recurring" --project P-XXXXXX
  zk quicknote "Quick observation" --author claude`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		text := args[0]
		authorFlag, _ := cmd.Flags().GetString("author")

		// Derive title: truncate at 50 runes on word boundary.
		title := text
		if utf8.RuneCountInString(title) > 50 {
			title = string([]rune(title)[:50])
			if idx := strings.LastIndex(title, " "); idx > 20 {
				title = title[:idx]
			}
		}

		storePath := getStorePath(cmd)
		s := store.NewStore(storePath)

		// Resolve author: --author flag > config default_author > "user".
		author := authorFlag
		if author == "" {
			if cfg, err := s.LoadConfig(); err == nil && cfg.DefaultAuthor != "" {
				author = cfg.DefaultAuthor
			} else {
				author = "user"
			}
		}

		note := model.NewNote(title, text, []string{})
		note.Metadata.Author = author

		if flagProject != "" {
			note.ProjectID = flagProject
		}

		if err := s.CreateNote(note); err != nil {
			return fmt.Errorf("create note: %w", err)
		}

		return getFormatter().PrintNote(note)
	},
}


func init() {
	quicknoteCmd.Flags().String("author", "", "note author (e.g., claude, gemini, human)")
	rootCmd.AddCommand(quicknoteCmd)
}
