package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"runtime"
	"time"

	proxyproto "github.com/armon/go-proxyproto"
	flags "github.com/jessevdk/go-flags"
	"github.com/kazeburo/ppdp/proxy"
	"github.com/kazeburo/ppdp/upstream"
	"github.com/lestrrat-go/server-starter/listener"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

var (
	// Version :
	Version string
)

type cmdOpts struct {
	Version             bool          `short:"v" long:"version" description:"Show version"`
	Listen              string        `short:"l" long:"listen" default:"0.0.0.0:3000" description:"address to bind"`
	Upstream            string        `long:"upstream" required:"true" description:"upstream server: upstream-server:port"`
	ProxyConnectTimeout time.Duration `long:"proxy-connect-timeout" default:"60s" description:"timeout of connection to upstream"`
	ProxyProtocol       bool          `long:"proxy-protocol" description:"use proxy-proto for listen"`
	DumpTCP             uint64        `long:"dump-tcp" default:"0" description:"Dump TCP. 0 = disable, 1 = src to dest, 2 = both"`
}

func printVersion() {
	fmt.Printf(`ppdp %s
Compiler: %s %s
`,
		Version,
		runtime.Compiler,
		runtime.Version())
}

func main() {
	opts := cmdOpts{}
	psr := flags.NewParser(&opts, flags.Default)
	_, err := psr.Parse()
	if err != nil {
		os.Exit(1)
	}

	if opts.Version {
		printVersion()
		return
	}

	logger, _ := zap.NewProduction()

	u, err := upstream.New(opts.Upstream, logger)
	if err != nil {
		logger.Fatal("failed initialize upstream", zap.Error(err))
	}
	defer u.Stop()

	var listens []net.Listener
	listens, err = listener.ListenAll()
	if err != nil && err != listener.ErrNoListeningTarget {
		logger.Fatal("failed initialize listener", zap.Error(err))
	}
	if len(listens) < 1 {
		logger.Info("Start listen",
			zap.String("listen", opts.Listen),
		)
		l, err := net.Listen("tcp", opts.Listen)
		if err != nil {
			logger.Fatal("failed to listen", zap.Error(err))

		}
		listens = append(listens, l)
	}

	ctx := context.Background()
	eg, ctx := errgroup.WithContext(ctx)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for _, l := range listens {
		if opts.ProxyProtocol {
			l = &proxyproto.Listener{Listener: l}
		}
		eg.Go(func() error {
			p := proxy.New(l, u, opts.ProxyConnectTimeout, opts.DumpTCP, logger)
			return p.Start(ctx)
		})
	}
	if err := eg.Wait(); err != nil {
		defer cancel()
		logger.Fatal("failed to start proxy", zap.Error(err))
	}
}
