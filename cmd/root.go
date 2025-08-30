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
		}

		count, err := finder.Run(ctx, os.Stdout, cfg)
		if err != nil {
			return err
		}
		if count == 0 {
			// useful in CI: non-zero when nothing matched
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
}
