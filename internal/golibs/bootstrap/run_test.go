package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"testing"
	"time"

	"go-boilerplate/internal/golibs/configs"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

type fakeConfig struct {
	configs.Base `mapstructure:",squash"`
}

type fakeServer struct {
	initErr      error
	bgRunning    chan struct{}
	shutdownDone bool
}

func (f *fakeServer) InitDependencies(context.Context, fakeConfig, *Resources) error {
	return f.initErr
}
func (f *fakeServer) RegisterGRPC(fakeConfig, *grpc.Server) {}
func (f *fakeServer) UnaryInterceptors(fakeConfig, *Resources) []grpc.UnaryServerInterceptor {
	return nil
}
func (f *fakeServer) RegisterHTTP(_ fakeConfig, r *gin.Engine) {
	r.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })
}
func (f *fakeServer) RunBackground(ctx context.Context, _ fakeConfig, _ *Resources) error {
	close(f.bgRunning)
	<-ctx.Done()
	return nil
}
func (f *fakeServer) Shutdown(context.Context) { f.shutdownDone = true }

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func setPorts(t *testing.T) (grpcPort, httpPort, metricsPort int) {
	grpcPort, httpPort, metricsPort = freePort(t), freePort(t), freePort(t)
	t.Setenv("COMMON_GRPC_PORT", strconv.Itoa(grpcPort))
	t.Setenv("COMMON_HTTP_PORT", strconv.Itoa(httpPort))
	t.Setenv("COMMON_METRICS_PORT", strconv.Itoa(metricsPort))
	return
}

func waitHTTP(t *testing.T, url string) *http.Response {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		resp, err := http.Get(url)
		if err == nil {
			return resp
		}
		if time.Now().After(deadline) {
			t.Fatalf("GET %s: %v", url, err)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestRun_StartsAllServersAndStopsGracefully(t *testing.T) {
	grpcPort, httpPort, metricsPort := setPorts(t)
	s := &fakeServer{bgRunning: make(chan struct{})}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- run[fakeConfig](ctx, "fake", "testdata", s) }()

	if resp := waitHTTP(t, fmt.Sprintf("http://127.0.0.1:%d/ping", httpPort)); resp.StatusCode != http.StatusOK {
		t.Fatalf("http /ping = %d", resp.StatusCode)
	}
	if resp := waitHTTP(t, fmt.Sprintf("http://127.0.0.1:%d/readyz", metricsPort)); resp.StatusCode != http.StatusOK {
		t.Fatalf("/readyz = %d", resp.StatusCode)
	}

	conn, err := grpc.NewClient(fmt.Sprintf("127.0.0.1:%d", grpcPort), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	hc, err := healthpb.NewHealthClient(conn).Check(context.Background(), &healthpb.HealthCheckRequest{})
	if err != nil || hc.Status != healthpb.HealthCheckResponse_SERVING {
		t.Fatalf("grpc health: %v %v", hc, err)
	}

	select {
	case <-s.bgRunning:
	case <-time.After(time.Second):
		t.Fatal("background worker not started")
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("run returned %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("run did not stop")
	}
	if !s.shutdownDone {
		t.Fatal("Shutdown hook not called")
	}
}

func TestRun_InitErrorStopsBeforeListening(t *testing.T) {
	_, httpPort, _ := setPorts(t)
	s := &fakeServer{initErr: errors.New("boom"), bgRunning: make(chan struct{})}

	err := run[fakeConfig](context.Background(), "fake", "testdata", s)
	if err == nil || !errors.Is(err, s.initErr) {
		t.Fatalf("want init error, got %v", err)
	}
	if _, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/ping", httpPort)); err == nil {
		t.Fatal("http port must not be opened when init fails")
	}
}

func TestRun_PortInUse(t *testing.T) {
	_, httpPort, _ := setPorts(t)
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", httpPort))
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	err = run[fakeConfig](context.Background(), "fake", "testdata", &fakeServer{bgRunning: make(chan struct{})})
	if err == nil {
		t.Fatal("expected listen error")
	}
}
