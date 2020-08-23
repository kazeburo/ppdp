package proxy

import (
	"context"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/kazeburo/ppdp/dumper"
	"github.com/kazeburo/ppdp/upstream"
	"go.uber.org/zap"
)

const (
	toUpstream   uint = 1
	fromUpstream uint = 2
	bufferSize        = 0xFFFF
)

// Proxy proxy struct
type Proxy struct {
	listener net.Listener
	upstream *upstream.Upstream
	timeout  time.Duration
	done     chan struct{}
	logger   *zap.Logger
	dumpTCP  uint64
	dumpPing bool
	maxRetry int
}

// New create new proxy
func New(l net.Listener, u *upstream.Upstream, t time.Duration, dumpTCP uint64, dumpPing bool, maxRetry int, logger *zap.Logger) *Proxy {
	return &Proxy{
		listener: l,
		upstream: u,
		timeout:  t,
		done:     make(chan struct{}),
		logger:   logger,
		dumpTCP:  dumpTCP,
		dumpPing: dumpPing,
		maxRetry: maxRetry,
	}
}

// Start start new proxy
func (p *Proxy) Start(ctx context.Context) error {
	wg := &sync.WaitGroup{}
	defer func() {
		wg.Wait()
		p.logger.Info("Complete shutdown",
			zap.String("listen", p.listener.Addr().String()),
		)
	}()
	go func() {
		select {
		case <-ctx.Done():
			p.logger.Info("Go shutdown",
				zap.String("listen", p.listener.Addr().String()),
			)
			p.listener.Close()
		}
	}()

	for {
		conn, err := p.listener.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok {
				if ne.Temporary() {
					p.logger.Warn("Failed to accept", zap.Error(err))
					continue
				}
			}
			if strings.Contains(err.Error(), "use of closed network connection") {
				select {
				case <-ctx.Done():
					return nil
				default:
					// fallthrough
				}
			}
			p.logger.Error("Failed to accept", zap.Error(err))
			return err
		}

		wg.Add(1)
		go func(c net.Conn) {
			defer wg.Done()
			p.handleConn(c)
		}(conn)
	}
}

func (p *Proxy) handleConn(c net.Conn) error {
	readLen := int64(0)
	writeLen := int64(0)
	hasError := false

	logger := p.logger.With(
		// zap.Uint64("seq", h.sq.Next()),
		zap.String("listener", p.listener.Addr().String()),
		zap.String("remote-addr", c.RemoteAddr().String()),
	)

	logger.Info("log", zap.String("status", "Connected"))

	ips, err := p.upstream.GetN(p.maxRetry, c.RemoteAddr())
	if err != nil {
		logger.Error("Failed to get upstream", zap.Error(err))
		c.Close()
		return err
	}

	var s net.Conn
	var ip *upstream.IP
	for _, ip = range ips {
		s, err = net.DialTimeout("tcp", ip.Address, p.timeout)
		if err == nil {
			break
		} else {
			logger.Warn("Failed to connect backend", zap.Error(err))
		}
	}
	if err != nil {
		logger.Error("Giveup to connect backends", zap.Error(err))
		c.Close()
		hasError = true
		return err
	}

	logger = logger.With(zap.String("upstream", ip.Address))
	dr := dumper.New(toUpstream, p.dumpPing, logger)
	ds := dumper.New(fromUpstream, p.dumpPing, logger)

	p.upstream.Use(ip)
	defer func() {
		dr.Stop()
		ds.Stop()
		p.upstream.Release(ip)
		status := "Suceeded"
		if hasError {
			status = "Failed"
		}
		logger.Info("log",
			zap.String("status", status),
			zap.Int64("read", readLen),
			zap.Int64("write", writeLen),
		)
	}()

	doneCh := make(chan bool)
	goClose := false

	// client => upstream
	go func() {
		defer func() { doneCh <- true }()
		s2 := s.(io.Writer)
		if p.dumpTCP > 0 {
			s2 = io.MultiWriter(s, dr)
		}
		n, err := io.Copy(s2, c)
		if err != nil {
			if !goClose {
				p.logger.Error("Copy from client", zap.Error(err))
				hasError = true
				return
			}
		}
		readLen += n
		return
	}()

	// upstream => client
	go func() {
		defer func() { doneCh <- true }()
		c2 := c.(io.Writer)
		if p.dumpTCP > 1 {
			c2 = io.MultiWriter(c, ds)
		}
		n, err := io.Copy(c2, s)
		if err != nil {
			if !goClose {
				p.logger.Error("Copy from upstream", zap.Error(err))
				hasError = true
				return
			}
		}
		writeLen += n
		return
	}()

	<-doneCh
	goClose = true
	s.Close()
	c.Close()
	<-doneCh
	return nil
}
