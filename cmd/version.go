package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

var Version = "dev"

func SetVersion(v string) {
	Version = v
}

func init() {
	rootCmd.AddCommand(versionCommand)
}

var versionCommand = &cobra.Command{
	Use:   "version",
	Short: "Print version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Version: %s  Author: 0x0c  Discord: @choree\n", Version)
	},
}
