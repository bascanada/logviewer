package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	sha1ver string
)

var versionCommand = &cobra.Command{
	Use:   "version",
	Short: "Display application version",
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Println(sha1ver)
	},
}

func init() {
	if sha1ver == "" {
		sha1ver = "develop"
	}
}
