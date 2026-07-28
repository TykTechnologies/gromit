package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/TykTechnologies/gromit/dhivex"
	"github.com/spf13/cobra"
)

var dhiVEXOptions dhivex.Options

var dhiVEXCmd = &cobra.Command{
	Use:   "dhi-vex IMAGE",
	Short: "Run Trivy with Docker's current signed DHI VEX",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dhiVEXOptions.Image = args[0]
		dhiVEXOptions.GitHubToken = os.Getenv("GITHUB_TOKEN")
		dhiVEXOptions.Log = cmd.ErrOrStderr()

		summary, err := dhivex.Scan(cmd.Context(), dhiVEXOptions)
		if err != nil {
			return err
		}
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(summary); err != nil {
			return fmt.Errorf("write DHI VEX summary: %w", err)
		}
		if summary.ActiveFindings > 0 {
			return fmt.Errorf("%d selected-severity finding(s) remain active", summary.ActiveFindings)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(dhiVEXCmd)
	flags := dhiVEXCmd.Flags()
	flags.StringVar(&dhiVEXOptions.Platform, "platform", "linux/amd64", "container platform to scan")
	flags.StringVar(&dhiVEXOptions.Severity, "severity", "HIGH,CRITICAL", "comma-separated Trivy severities")
	flags.StringVar(&dhiVEXOptions.OutputDir, "output-dir", "trivy-dhi-vex-report", "new directory for atomic scan evidence")
	flags.StringVar(&dhiVEXOptions.Timeout, "timeout", "45m", "timeout for each Trivy image operation")
	flags.StringVar(&dhiVEXOptions.AdvisoriesRef, "advisories-ref", "main", "Docker DHI advisories branch, tag, or commit")
	flags.StringVar(&dhiVEXOptions.TrivyPath, "trivy", "trivy", "Trivy executable")
	flags.StringVar(&dhiVEXOptions.CosignPath, "cosign", "cosign", "Cosign executable")
}
