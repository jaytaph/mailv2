package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

var rootCmd = &cobra.Command{
	Use:   "bm-client",
	Short: "BitMaelum client",
	Long: `This client allows you to manage accounts, read and compose mail.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		fmt.Println("")
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringP("config", "c", "", "configuration file")
	rootCmd.PersistentFlags().StringP("password", "p", "", "password to unlock your account vault")
}
