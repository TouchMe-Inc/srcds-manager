package cmd

import (
	"fmt"
	"github.com/keyCat/srcds-manager/config"
	"github.com/spf13/cobra"
	"log"
)

var rootCmd = &cobra.Command{
	Use:   "servers",
	Short: "Manage srcds servers",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		config.ReadOrThrow()
	},
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	Long: fmt.Sprintf(`This command helps to manage the state of multiple running servers by facilitating tasks of starting, restarting and updating servers.

Version: %s  Author: 0x0c  Discord: @choree`, Version),
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		log.Fatalf("%v\n", err)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&config.Path, "config", "c", config.GetDefaultLocation(), "config file path")
}
