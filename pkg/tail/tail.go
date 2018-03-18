package tail

import (
	"bufio"
	"context"
	"log"
	"os"
	"sync"

	"github.com/marcsauter/rtail/pkg/pb"
)

// Tailer interface
type Tailer interface {
	File(context.Context, *pb.FileRequest)
}

// tail
type tail struct {
	sync.Mutex
	wg     *sync.WaitGroup
	stream pb.Proxy_RegisterClient
}

// New returns a new Tailer
func New(wg *sync.WaitGroup, stream pb.Proxy_RegisterClient) Tailer {
	return &tail{
		wg:     wg,
		stream: stream,
	}
}

// send serializes send operation on gRPC stream
func (t *tail) send(line *pb.Line) error {
	t.Lock()
	defer t.Unlock()
	return t.stream.Send(&pb.AgentMessage{
		Line: line,
	})
}

// File tail a file
func (t *tail) File(ctx context.Context, r *pb.FileRequest) {
	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		f, _ := os.Open(r.Path)
		defer f.Close()
		scanner := bufio.NewScanner(f)
		scanner.Split(bufio.ScanLines)
		for scanner.Scan() {
			if err := t.send(&pb.Line{
				Line: scanner.Text(),
				Key:  r.Key,
			}); err != nil {
				log.Printf("send failed: %v", err)
				return
			}
		}
		if err := scanner.Err(); err != nil {
			log.Printf("reading %s failed: %v", r.Path, err)
		}
		if err := t.send(&pb.Line{
			Eof: true,
			Key: r.Key,
		}); err != nil {
			log.Printf("send failed: %v", err)
			return
		}
	}()
}
