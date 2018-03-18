package cmd

import (
	"context"
	"io"
	"log"
	"os"
	"sync"

	"github.com/marcsauter/rtail/pkg/global"
	"github.com/marcsauter/rtail/pkg/pb"
	"github.com/marcsauter/rtail/pkg/tail"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// agentCmd represents the agent command
var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		paddr, _ := cmd.Flags().GetString("proxy")
		// TODO: globs
		//globs := []string{}
		conn, err := grpc.Dial(paddr, grpc.WithInsecure())
		if err != nil {
			log.Fatalf("could not connect proxy %s: %v", paddr, err)
		}
		defer conn.Close()
		// metadata - provider name
		hostname, err := os.Hostname()
		if err != nil {
			log.Fatal(err)
		}
		ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs(
			"provider", hostname,
		))
		// register provider
		c := pb.NewProxyClient(conn)
		stream, err := c.Register(ctx)
		if err != nil {
			log.Printf("register failed: %v", err)
			return
		}
		var wg sync.WaitGroup
		for {
			log.Printf("awaiting request\n")
			r, err := stream.Recv()
			switch err {
			case io.EOF:
				log.Printf("proxy stream closed")
				return
			case nil:
				log.Printf("request for %s received\n", r.Path)
				tail.New(&wg, stream).File(ctx, r)
			default:
				log.Printf("failed to receive request from proxy: %v", err)
				return
			}
		}
		wg.Wait()
	},
}

func init() {
	rootCmd.AddCommand(agentCmd)

	agentCmd.Flags().StringP("proxy", "p", global.ProxyDefault, "proxy address")
}
