package cli

import (
	"os"

	"github.com/fatih/color"
	"github.com/gptscript-ai/cmd"
	"github.com/njhale/maskfs/pkg/logger"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

type MaskFS struct {
	Debug bool `usage:"Enable debug logging"`
}

func (m *MaskFS) PersistentPre(*cobra.Command, []string) error {
	if os.Getenv("NO_COLOR") != "" || !term.IsTerminal(int(os.Stdout.Fd())) {
		color.NoColor = true
	}

	if m.Debug {
		logger.SetDebug()
	}

	return nil
}

func New() *cobra.Command {
	root := &MaskFS{}
	return cmd.Command(root,
		&Server{},
	)
}

func (a *MaskFS) Run(cmd *cobra.Command, _ []string) error {
	return cmd.Help()
}
