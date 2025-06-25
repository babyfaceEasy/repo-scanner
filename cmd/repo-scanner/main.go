package main

import (
	"fmt"
	"os"

	"github.com/babyfaceeasy/repo-scanner/internal/config"
	"github.com/babyfaceeasy/repo-scanner/internal/github"
	"github.com/babyfaceeasy/repo-scanner/internal/output"
	"github.com/babyfaceeasy/repo-scanner/internal/scanner"
	"github.com/babyfaceeasy/repo-scanner/internal/service"
	"github.com/babyfaceeasy/repo-scanner/pkg/logger"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func main() {
	// load .env file
	if err := godotenv.Load(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load .env file: %v\n", err)
	}

	// initialize logger
	log, err := logger.New()
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
			svc := service.New(
				config.New(),
				github.NewClient(os.Getenv("GITHUB_TOKEN"), log),
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
