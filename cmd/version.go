package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
    sha1ver   string
)


var versionCommand = &cobra.Command{
    Use: "version",
    Short: "Display application version",
    Run: func(cnd *cobra.Command, args []string) {
        fmt.Println(sha1ver)
    },
}

func init() {
    if sha1ver == "" {
        sha1ver = "develop"
    }
}
