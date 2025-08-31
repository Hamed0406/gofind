package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/Hamed0406/gofind/internal/finder"
	"github.com/spf13/cobra"
)

var (
	flagPath         string
	flagGitignore    bool
	flagExtraIgnores []string

	// filters
	flagExts   []string
	flagName   string
	flagRegex  string
	flagType   string // f|d|a
	flagHidden bool

	// NEW:
	flagLarger  string
	flagSmaller string
	flagSince   string
	flagOutput  string // path|json|ndjson
)

var rootCmd = &cobra.Command{
	Use:   "gofind",
	Short: "Fast cross-platform file finder with smart ignores",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()

		cfg := finder.Config{
			Path:             flagPath,
			RespectGitignore: flagGitignore,
			ExtraIgnores:     flagExtraIgnores,

			Exts:   flagExts,
			Name:   flagName,
			Regex:  flagRegex,
			Type:   flagType,
			Hidden: flagHidden,

			// NEW:
			Larger:  flagLarger,
			Smaller: flagSmaller,
			Since:   flagSince,
			Output:  flagOutput,
		}

		count, err := finder.Run(ctx, os.Stdout, cfg)
		if err != nil {
			return err
		}
		if count == 0 {
			os.Exit(1)
		}
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringVarP(&flagPath, "path", "p", ".", "start directory")
	rootCmd.Flags().BoolVar(&flagGitignore, "respect-gitignore", true, "respect .gitignore files")
	rootCmd.Flags().StringSliceVar(&flagExtraIgnores, "ignore", nil, "extra ignore patterns (repeatable)")

	// filters
	rootCmd.Flags().StringSliceVar(&flagExts, "ext", nil, "filter by extension (repeatable), e.g. --ext .go --ext .yaml")
	rootCmd.Flags().StringVar(&flagName, "name", "", "substring match on filename")
	rootCmd.Flags().StringVar(&flagRegex, "regex", "", "regular expression on filename")
	rootCmd.Flags().StringVar(&flagType, "type", "f", "entry type: f=files, d=dirs, a=all")
	rootCmd.Flags().BoolVar(&flagHidden, "hidden", false, "include hidden files (dotfiles)")

	// NEW:
	rootCmd.Flags().StringVar(&flagLarger, "larger", "", "size greater than (e.g. 100K, 20M, 1G)")
	rootCmd.Flags().StringVar(&flagSmaller, "smaller", "", "size less than (e.g. 1M)")
	rootCmd.Flags().StringVar(&flagSince, "since", "", "modified since (e.g. 7d, 3h, 2025-08-01)")
	rootCmd.Flags().StringVar(&flagOutput, "output", "path", "output format: path|json|ndjson")
}
