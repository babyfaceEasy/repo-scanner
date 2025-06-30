package main

import (
	"fmt"
	"os"
	"time"

	"github.com/babyfaceeasy/repo-scanner/internal/config"
	"github.com/babyfaceeasy/repo-scanner/internal/env"
	"github.com/babyfaceeasy/repo-scanner/internal/github"
	"github.com/babyfaceeasy/repo-scanner/internal/output"
	"github.com/babyfaceeasy/repo-scanner/internal/retry"
	"github.com/babyfaceeasy/repo-scanner/internal/scanner"
	"github.com/babyfaceeasy/repo-scanner/internal/service"
	"github.com/babyfaceeasy/repo-scanner/pkg/logger"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func main() {
	// load environment variables
	cfg, err := env.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load environment variables: %v\n", err)
	}

	// initialize logger
	log, err := logger.New(cfg.LogEnv)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	rootCmd := &cobra.Command{
		Use:   "repo-scanner",
		Short: "A CLI tool to scan GitHub repositories for large files",
	}

	scanCmd := &cobra.Command{
		Use:   "scan [json-config]",
		Short: "Scan a repository for files larger than a specified size",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			githubClient := github.NewClient(cfg.GitHubToken, log)
			// TODO: the variables been passed here can be converted to env variables.
			retryClient := retry.NewRetrier(githubClient, log, 3, 1*time.Second, 15*time.Second)
			svc := service.New(
				config.New(),
				retryClient,
				scanner.New(log),
				output.New(),
				log,
			)

			if err := svc.Scan(args[0]); err != nil {
				log.Error("Scan failed", zap.Error(err))
				os.Exit(1)
			}
		},
	}

	rootCmd.AddCommand(scanCmd)
	if err := rootCmd.Execute(); err != nil {
		log.Error("Command execution failed", zap.Error(err))
		os.Exit(1)
	}
}
