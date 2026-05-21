package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/postfriday/gitlab-labelctl/internal/config"
	"github.com/postfriday/gitlab-labelctl/internal/diff"
	"github.com/postfriday/gitlab-labelctl/internal/export"
	"github.com/postfriday/gitlab-labelctl/internal/gitlab"
	"github.com/postfriday/gitlab-labelctl/internal/schema"
	"github.com/postfriday/gitlab-labelctl/internal/validate"
	"github.com/postfriday/gitlab-labelctl/pkg/version"
	"github.com/spf13/cobra"
)

var (
	cfgFile         string
	token           string
	tokenFile       string
	dryRun          bool
	verbose         bool
	jsonOutput      bool
	projectRef      string
	groupRef        string
	outputFile      string
	continueOnError bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "gitlab-labelctl",
		Short: "Declarative GitLab label management",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if cfgFile == "" {
				return fmt.Errorf("--config is required")
			}
			return nil
		},
	}

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "YAML configuration file")
	rootCmd.PersistentFlags().StringVar(&token, "token", "", "GitLab personal access token")
	rootCmd.PersistentFlags().StringVar(&tokenFile, "token-file", "", "File containing GitLab token")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Run without mutating GitLab")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Enable verbose logging")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Produce machine-readable JSON output")
	rootCmd.PersistentFlags().BoolVar(&continueOnError, "continue-on-error", false, "Continue on non-fatal errors")

	rootCmd.AddCommand(versionCmd())
	rootCmd.AddCommand(validateCmd())
	rootCmd.AddCommand(diffCmd())
	rootCmd.AddCommand(syncCmd())
	rootCmd.AddCommand(exportCmd())
	rootCmd.AddCommand(driftCmd())
	rootCmd.AddCommand(schemaCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func loadAppConfig(ctx context.Context) (*config.Config, error) {
	cfg, err := config.Load(ctx, cfgFile)
	if err != nil {
		return nil, err
	}
	authToken, err := config.ResolveToken(token, tokenFile, cfg)
	if err != nil {
		return nil, err
	}
	cfg.GitLab.Auth.Token = authToken
	return cfg, nil
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(version.Version)
		},
	}
}

func validateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate YAML configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			cfg, err := loadAppConfig(ctx)
			if err != nil {
				return err
			}
			if err := validate.Validate(ctx, cfg); err != nil {
				return err
			}
			return renderValidateSuccess(os.Stdout, cfg.Source, jsonOutput)
		},
	}
}

func renderValidateSuccess(out io.Writer, configPath string, jsonOutput bool) error {
	if jsonOutput {
		return json.NewEncoder(out).Encode(struct {
			Valid  bool   `json:"valid"`
			Config string `json:"config"`
		}{
			Valid:  true,
			Config: configPath,
		})
	}
	_, err := fmt.Fprintf(out, "Configuration is valid: %s\n", configPath)
	return err
}

func diffCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "diff",
		Short: "Show the label reconciliation diff",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			cfg, err := loadAppConfig(ctx)
			if err != nil {
				return err
			}
			client, err := gitlab.NewClient(cfg)
			if err != nil {
				return err
			}
			if err := validate.Validate(ctx, cfg); err != nil {
				return err
			}
			results, err := diff.ComputeDiff(ctx, cfg, client)
			if err != nil {
				return err
			}
			diff.Render(results, os.Stdout, jsonOutput)
			return nil
		},
	}
}

func syncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Reconcile GitLab labels against desired state",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			cfg, err := loadAppConfig(ctx)
			if err != nil {
				return err
			}
			client, err := gitlab.NewClient(cfg)
			if err != nil {
				return err
			}
			if err := validate.Validate(ctx, cfg); err != nil {
				return err
			}
			results, err := diff.ComputeDiff(ctx, cfg, client)
			if err != nil {
				return err
			}
			if dryRun || cfg.Defaults.DryRun {
				diff.Render(results, os.Stdout, jsonOutput)
				return nil
			}
			return diff.Reconcile(ctx, results, client, continueOnError)
		},
	}
}

func exportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export GitLab labels to YAML",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			cfg, err := loadAppConfig(ctx)
			if err != nil {
				return err
			}
			if projectRef == "" && groupRef == "" {
				return fmt.Errorf("--project or --group is required")
			}
			client, err := gitlab.NewClient(cfg)
			if err != nil {
				return err
			}
			var out *os.File
			if outputFile == "" {
				out = os.Stdout
			} else {
				f, err := os.Create(outputFile)
				if err != nil {
					return err
				}
				defer f.Close()
				out = f
			}
			return export.Run(ctx, client, projectRef, groupRef, out)
		},
	}
	cmd.Flags().StringVar(&projectRef, "project", "", "Project path or ID to export labels from")
	cmd.Flags().StringVar(&groupRef, "group", "", "Group path or ID to export labels from")
	cmd.Flags().StringVar(&outputFile, "output", "", "Write exported YAML to this file")
	return cmd
}

func driftCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "drift",
		Short: "Detect drift between GitLab and desired state",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			cfg, err := loadAppConfig(ctx)
			if err != nil {
				return err
			}
			client, err := gitlab.NewClient(cfg)
			if err != nil {
				return err
			}
			if err := validate.Validate(ctx, cfg); err != nil {
				return err
			}
			results, err := diff.ComputeDiff(ctx, cfg, client)
			if err != nil {
				return err
			}
			diff.Render(results, os.Stdout, jsonOutput)
			return nil
		},
	}
}

func schemaCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "schema",
		Short: "Print JSON schema for YAML configs",
		RunE: func(cmd *cobra.Command, args []string) error {
			data, err := schema.Contents()
			if err != nil {
				return err
			}
			_, err = os.Stdout.Write(data)
			return err
		},
	}
}
