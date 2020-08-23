package upstream

import (
	"context"
	"math/rand"
	"net"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/stathat/consistent"
	"go.uber.org/zap"
)

// Upstream struct
type Upstream struct {
	port       string
	host       string
	ips        []*IP
	iph        map[string]*IP
	csum       string
	consistent *consistent.Consistent
	balancing  string
	logger     *zap.Logger
	mu         sync.Mutex
	// current resolved record version
	version uint64
	cancel  context.CancelFunc
}

// IP : IP with counter
type IP struct {
	Address string
	// # requerst in busy
	busy int64
	// resolved record version
	version uint64
}

// String :
func (ip IP) String() string {
	return ip.Address
}

// New :
func New(upstream, balancing string, logger *zap.Logger) (*Upstream, error) {
	hostPortSplit := strings.Split(upstream, ":")
	h := hostPortSplit[0]
	p := ""
	if len(hostPortSplit) > 1 {
		p = hostPortSplit[1]
	}

	ctx, cancel := context.WithCancel(context.Background())

	um := &Upstream{
		host:      h,
		port:      p,
		version:   0,
		balancing: balancing,
		logger:    logger,
		cancel:    cancel,
	}

	ips, err := um.RefreshIP(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed initial resolv hostname")
	}
	if len(ips) < 1 {
		return nil, errors.New("Could not resolv hostname")
	}
	go um.Run(ctx)
	return um, nil
}

// RefreshIP : resolve hostname
func (u *Upstream) RefreshIP(ctx context.Context) ([]*IP, error) {
	u.mu.Lock()
	u.version++
	u.mu.Unlock()

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, u.host)
	cancel()
	if err != nil {
		return nil, err
	}

	sort.Slice(addrs, func(i, j int) bool {
		return addrs[i].IP.String() > addrs[j].IP.String()
	})

	csumTexts := make([]string, len(addrs))
	ips := make([]*IP, len(addrs))
	iph := map[string]*IP{}

	consistent := consistent.New()

	for i, ia := range addrs {
		csumTexts[i] = ia.IP.String()
		address := ia.IP.String()
		if u.port != "" {
			address = address + ":" + u.port
		}
		ipa := &IP{
			Address: address,
			version: u.version,
			busy:    0,
		}
		ips[i] = ipa
		iph[address] = ipa
		consistent.Add(address)
	}

	csum := strings.Join(csumTexts, ",")
	u.mu.Lock()
	defer u.mu.Unlock()
	if csum != u.csum {
		u.csum = csum
		u.ips = ips
		u.iph = iph
		u.consistent = consistent
	}

	return ips, nil
}

// Run : resolv hostname in background
func (u *Upstream) Run(ctx context.Context) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case _ = <-ticker.C:
			_, err := u.RefreshIP(ctx)
			if err != nil {
				u.logger.Error("failed refresh ip", zap.Error(err))
			}
		}
	}
}

// GetN :
func (u *Upstream) GetN(maxIP int, src net.Addr) ([]*IP, error) {
	u.mu.Lock()
	defer u.mu.Unlock()

	if len(u.ips) < 1 {
		return nil, errors.New("No upstream hosts")
	}

	hostPortSplit := strings.Split(src.String(), ":")
	srcAddr := hostPortSplit[0]

	switch u.balancing {
	case "fixed":
		return u.getNByHash(maxIP, u.host)
	case "iphash":
		return u.getNByHash(maxIP, srcAddr)
	case "remotehash":
		return u.getNByHash(maxIP, src.String())
	default:
		return u.getNByLC(maxIP)
	}
}

func (u *Upstream) getNByHash(maxIP int, key string) ([]*IP, error) {
	if len(u.ips) < maxIP {
		maxIP = len(u.ips)
	}

	ips := make([]*IP, 0, maxIP)

	res, err := u.consistent.GetN(key, maxIP)
	if err != nil {
		return ips, err
	}

	for _, ip := range res {
		ipa, ok := u.iph[ip]
		if !ok {
			continue
		}
		ips = append(ips, ipa)
		if len(ips) == maxIP {
			break
		}
	}

	return ips, nil

}

func (u *Upstream) getNByLC(maxIP int) ([]*IP, error) {

	sort.Slice(u.ips, func(i, j int) bool {
		if u.ips[i].busy == u.ips[j].busy {
			return rand.Intn(2) == 0
		}
		return u.ips[i].busy < u.ips[j].busy
	})

	if len(u.ips) < maxIP {
		maxIP = len(u.ips)
	}

	ips := make([]*IP, 0, maxIP)
	for _, ipa := range u.ips {
		ips = append(ips, ipa)
		if len(ips) == maxIP {
			break
		}
	}

	return ips, nil
}

// Use : Increment counter
func (u *Upstream) Use(o *IP) {
	u.mu.Lock()
	defer u.mu.Unlock()
	o.busy = o.busy + 1
}

// Release : decrement counter
func (u *Upstream) Release(o *IP) {
	u.mu.Lock()
	defer u.mu.Unlock()
	o.busy = o.busy - 1
}

// Stop : stop upstream updater
func (u *Upstream) Stop() {
	u.cancel()
}
