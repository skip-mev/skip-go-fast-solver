/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "solvercli",
	Short: "CLI for solver functionality",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.solvercli.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	rootCmd.PersistentFlags().String("config", "./config/local/config.yml", "path to solver config file")
	rootCmd.PersistentFlags().String("keys", "./config/local/keys.json", "path to solver key file. must be specified if key-store-type is plaintext-file or encrpyted-file")
	rootCmd.PersistentFlags().String("key-store-type", "plaintext-file", "where to load the solver keys from. (plaintext-file, encrypted-file, env)")
	rootCmd.PersistentFlags().String("aes-key-hex", "", "hex-encoded AES key used to decrypt keys file. must be specified if key-store-type is encrypted-file")
	rootCmd.PersistentFlags().String("sqlite-db-path", "./solver.db", "path to sqlite db file")
	rootCmd.PersistentFlags().String("migrations-path", "./db/migrations", "path to db migrations directory")
}
