package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/synclet-io/synclet/pkg/coordinator"
)

func newOrchestrateCommand() *cobra.Command {
	var (
		jobID        string
		connectionID string
		serverAddr   string
		dataDir      string

		sourceID    string
		sourceImage string
		sourceCmd   string

		destID    string
		destImage string
		destCmd   string

		// Namespace/prefix rewriting
		namespaceDef          string
		customNamespaceFormat string
		streamPrefix          string

		// Secrets dir (K8s Secret volume mount)
		secretsDir string

		// Task mode flags
		taskMode bool
		taskID   string
		taskType string
	)

	cmd := &cobra.Command{
		Use:    "_orchestrate",
		Short:  "Run the in-pod coordinator process (internal use only)",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Task mode: simplified coordinator for connector operations (check/spec/discover).
			if taskMode {
				if taskID == "" {
					return fmt.Errorf("--task-id is required in task mode")
				}

				if taskType == "" {
					return fmt.Errorf("--task-type is required in task mode")
				}

				if serverAddr == "" {
					return fmt.Errorf("--server-addr is required in task mode")
				}

				return coordinator.RunTask(cmd.Context(), coordinator.TaskConfig{
					TaskID:     taskID,
					TaskType:   taskType,
					ServerAddr: serverAddr,
					DataDir:    dataDir,
				})
			}

			// Sync mode: validate required sync flags.
			if jobID == "" {
				return fmt.Errorf("--job-id is required in sync mode")
			}

			if connectionID == "" {
				return fmt.Errorf("--connection-id is required in sync mode")
			}

			if serverAddr == "" {
				return fmt.Errorf("--server-addr is required in sync mode")
			}

			if sourceImage == "" {
				return fmt.Errorf("--source-image is required in sync mode")
			}

			if destImage == "" {
				return fmt.Errorf("--dest-image is required in sync mode")
			}

			if secretsDir == "" {
				return fmt.Errorf("--secrets-dir is required in sync mode")
			}

			cfg := coordinator.Config{
				JobID:                 jobID,
				ConnectionID:          connectionID,
				ServerAddr:            serverAddr,
				DataDir:               dataDir,
				SecretsDir:            secretsDir,
				SourceID:              sourceID,
				DestinationID:         destID,
				SourceImage:           sourceImage,
				SourceCommand:         parseCommand(sourceCmd),
				DestImage:             destImage,
				DestCommand:           parseCommand(destCmd),
				NamespaceDefinition:   namespaceDef,
				CustomNamespaceFormat: customNamespaceFormat,
				StreamPrefix:          streamPrefix,
			}

			return coordinator.Run(cmd.Context(), cfg)
		},
	}

	// Shared flags (both modes)
	cmd.Flags().StringVar(&serverAddr, "server-addr", "", "Synclet server address (e.g. http://synclet:8080)")
	cmd.Flags().StringVar(&dataDir, "data-dir", "/shared", "Shared data directory")
	cmd.Flags().StringVar(&secretsDir, "secrets-dir", "", "Path to mounted secrets volume (K8s Secret mount)")

	// Sync mode flags
	cmd.Flags().StringVar(&jobID, "job-id", "", "Sync job ID (sync mode)")
	cmd.Flags().StringVar(&connectionID, "connection-id", "", "Connection ID (sync mode)")
	cmd.Flags().StringVar(&sourceID, "source-id", "", "Source entity UUID")
	cmd.Flags().StringVar(&destID, "dest-id", "", "Destination entity UUID")
	cmd.Flags().StringVar(&sourceImage, "source-image", "", "Source connector image")
	cmd.Flags().StringVar(&sourceCmd, "source-cmd", "read --config /shared/source-config.json --catalog /shared/source-catalog.json", "Source command")
	cmd.Flags().StringVar(&destImage, "dest-image", "", "Destination connector image")
	cmd.Flags().StringVar(&destCmd, "dest-cmd", "write --config /shared/dest-config.json --catalog /shared/dest-catalog.json", "Dest command")

	// Namespace/prefix rewriting flags
	cmd.Flags().StringVar(&namespaceDef, "namespace-definition", "", "Namespace definition: source, destination, custom")
	cmd.Flags().StringVar(&customNamespaceFormat, "custom-namespace-format", "", "Custom namespace format template")
	cmd.Flags().StringVar(&streamPrefix, "stream-prefix", "", "Stream name prefix")

	// Task mode flags
	cmd.Flags().BoolVar(&taskMode, "task-mode", false, "Run in connector task mode (check/spec/discover)")
	cmd.Flags().StringVar(&taskID, "task-id", "", "Connector task ID (task mode only)")
	cmd.Flags().StringVar(&taskType, "task-type", "", "Task type: Check, Spec, Discover (task mode only)")

	return cmd
}

func parseCommand(cmd string) []string {
	return strings.Fields(cmd)
}
