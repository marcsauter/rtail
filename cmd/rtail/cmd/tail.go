package cmd

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/marcsauter/rtail/pkg/global"
	"github.com/marcsauter/rtail/pkg/pb"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// tailCmd represents the tail command
var tailCmd = &cobra.Command{
	Use:   "tail file@host",
	Short: "tail",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		// validate argument(s)
		if len(args) != 1 {
			cmd.Usage()
			return
		}
		filenode := strings.Split(args[0], "@")
		if len(filenode) != 2 {
			cmd.Usage()
			return
		}
		// evaluate flags
		proxy, _ := cmd.Flags().GetString("proxy")
		timeout, _ := cmd.Flags().GetDuration("timeout")
		last, _ := cmd.Flags().GetInt("last")
		follow, _ := cmd.Flags().GetBool("follow")
		// connect to proxy
		conn, err := grpc.Dial(proxy, grpc.WithInsecure())
		if err != nil {
			log.Fatalf("did not connect: %v", err)
		}
		defer conn.Close()
		// context
		ctxt, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		// add token to metadata
		token := os.Getenv("TOKEN")
		ctxm := metadata.NewOutgoingContext(ctxt, metadata.Pairs(
			"token", token,
		))
		c := pb.NewProxyClient(conn)
		stream, err := c.Get(ctxm, &pb.FileRequest{
			Provider: filenode[1],
			Path:     filenode[0],
			Last:     uint32(last),
			Follow:   follow,
		})
		if err != nil {
			log.Fatalf("could not tail %s: %v", args[0], err)
		}
		for {
			l, err := stream.Recv()
			switch err {
			case io.EOF:
				return
			case nil:
				fmt.Println(l.Line)
			default:
				log.Fatalf("failed to receive a line from %s: %v", args[0], err)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(tailCmd)

	tailCmd.Flags().StringP("proxy", "p", global.ProxyDefault, "proxy address")
	tailCmd.Flags().DurationP("timeout", "t", 5*time.Second, "timeout")
	tailCmd.Flags().IntP("last", "l", 0, "last lines (0 = whole file)")
	tailCmd.Flags().BoolP("follow", "f", false, "follow")
}
