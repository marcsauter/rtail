package proxy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"sync"

	"github.com/marcsauter/rtail/pkg/pb"
	uuid "github.com/satori/go.uuid"
	"google.golang.org/grpc/metadata"
)

// Server implements interface pb.ProxyServer
type Server struct {
	sync.Mutex
	token     string
	providers map[string]provider
}

type provider struct {
	sync.Mutex
	globs     []string
	request   chan *pb.FileRequest
	callbacks map[string]callbackFunc
}

type callbackFunc func(*pb.Line)

// New returns a new proxy server
func New(token string) *Server {
	return &Server{
		token:     token,
		providers: make(map[string]provider),
	}
}

func (p *provider) addCallback(key string, c callbackFunc) {
	p.Lock()
	p.callbacks[key] = c
	p.Unlock()
}

func (p *provider) delCallback(key string) {
	p.Lock()
	delete(p.callbacks, key)
	p.Unlock()
}

// Register a new file provider and start a dedicated goroutine
func (s *Server) Register(srv pb.Proxy_RegisterServer) error {
	ctx := srv.Context()
	name, err := getMetadata(ctx, "provider")
	if err != nil {
		return err
	}
	p := provider{
		request:   make(chan *pb.FileRequest),
		callbacks: make(map[string]callbackFunc),
	}
	s.Lock()
	s.providers[name] = p
	s.Unlock()
	log.Printf("registered: %s\n", name)
	log.Printf("starting provider %s\n", name)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case r, ok := <-p.request:
				log.Printf("request for %s@%s received\n", r.Path, r.Provider)
				if !ok {
					log.Printf("request channel of provider %s closed", name)
					return
				}
				if err := srv.Send(r); err != nil {
					log.Printf("send failed: %v", err)
					return
				}
			}
		}
	}()
	for {
		m, err := srv.Recv()
		switch err {
		case io.EOF:
			return fmt.Errorf("provider %s closed", name)
		case nil:
			p.callbacks[m.Line.Key](m.Line)
		default:
			return fmt.Errorf("failed to receive from provider %s: %v", name, err)
		}
	}
	return nil
}

// Get a file via proxy
func (s *Server) Get(r *pb.FileRequest, stream pb.Proxy_GetServer) error {
	token, err := getMetadata(stream.Context(), "token")
	if err != nil {
		return err
	}
	if token != s.token {
		return fmt.Errorf("client %s is not allowed to request files", "TODO")
	}
	// add request key
	key := uuid.NewV4()
	r.Key = key.String()
	// find provider in registry
	p, ok := s.providers[r.Provider]
	if !ok {
		return fmt.Errorf("provider %s unknown", r.Provider)
	}
	log.Printf("requesting %s@%s\n", r.Path, r.Provider)
	// TODO: non-blocking?
	p.request <- r
	done := make(chan struct{})
	p.addCallback(r.Key, func(l *pb.Line) {
		if l.Eof {
			close(done)
			return
		}
		// TODO: handle error
		stream.Send(l)
	})
	<-done
	p.delCallback(r.Key)
	log.Printf("request %s@%s\n", r.Path, r.Provider)
	return nil
}

func getMetadata(ctx context.Context, key string) (string, error) {
	var empty string
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return empty, errors.New("could not retrieve metadata from context")
	}
	val, ok := md[key]
	if !ok || len(val) != 1 {
		return empty, fmt.Errorf("could not retrieve %s from metadata", key)
	}
	return val[0], nil
}
