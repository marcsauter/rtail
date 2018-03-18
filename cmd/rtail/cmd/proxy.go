package cmd

import (
	"log"
	"net"
	"os"

	"github.com/marcsauter/rtail/pkg/global"
	"github.com/marcsauter/rtail/pkg/pb"
	"github.com/marcsauter/rtail/pkg/proxy"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// proxyCmd represents the proxy command
var proxyCmd = &cobra.Command{
	Use:   "proxy",
	Short: "start proxy",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		laddr, _ := cmd.Flags().GetString("listen")
		l, err := net.Listen("tcp", laddr)
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}
		s := grpc.NewServer()
		token := os.Getenv("TOKEN")
		p := proxy.New(token)
		pb.RegisterProxyServer(s, p)
		// Register reflection service on gRPC server.
		reflection.Register(s)
		if err := s.Serve(l); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(proxyCmd)

	proxyCmd.Flags().StringP("listen", "l", global.ProxyDefault, "listener address and port")
}
