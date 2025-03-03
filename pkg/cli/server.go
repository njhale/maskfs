package cli

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/njhale/maskfs/pkg/server"
	"github.com/spf13/cobra"
)

type Server struct {
	server.Config
}

func (s *Server) Run(cmd *cobra.Command, _ []string) error {
	ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt, os.Kill, syscall.SIGTERM)
	defer cancel()
	return server.Run(ctx, s.Config)
}
