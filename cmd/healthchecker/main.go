package main

import (
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"github.com/streamingfast/logging"

	"github.com/spf13/cobra"
)

var zlog, _ = logging.ApplicationLogger("healthChecker", "github.com/streamingfast/blockmeta-service/server/cmd/healthChecker")

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func mustGetStringSlice(cmd *cobra.Command, flagName string) []string {
	val, err := cmd.Flags().GetStringSlice(flagName)
	if err != nil {
		panic(fmt.Sprintf("flags: couldn't find flag %q", flagName))
	}
	if len(val) == 0 {
		return nil
	}
	return val
}

func mustGetString(cmd *cobra.Command, flagName string) string {
	val, err := cmd.Flags().GetString(flagName)
	if err != nil {
		panic(fmt.Sprintf("flags: couldn't find flag %q", flagName))
	}
	return val
}

func mustGetHex(cmd *cobra.Command, flagName string) []byte {
	val, err := cmd.Flags().GetString(flagName)

	if err != nil {
		panic(fmt.Sprintf("flags: couldn't find flag %q", flagName))
	}

	bytes, err := hex.DecodeString(val)
	if err != nil {
		panic(fmt.Sprintf("flags: couldn't decode hex string %q", val))
	}

	return bytes
}

func mustGetDuration(cmd *cobra.Command, flagName string) time.Duration {
	val, err := cmd.Flags().GetDuration(flagName)
	if err != nil {
		panic(fmt.Sprintf("flags: couldn't find flag %q", flagName))
	}
	return val
}
