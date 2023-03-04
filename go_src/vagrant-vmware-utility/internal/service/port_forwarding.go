// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package service

import (
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"sync"

	hclog "github.com/hashicorp/go-hclog"

	"github.com/hashicorp/vagrant-vmware-desktop/go_src/vagrant-vmware-utility/settings"
)

type Forward struct {
	Active bool
	Ctx    context.Context
	Fwd    *settings.Forward

	cancel context.CancelFunc
	l      sync.Mutex
	logger hclog.Logger
}

func (f *Forward) Deactivate() error {
	f.l.Lock()
	defer f.l.Unlock()

	f.cancel()
	f.Active = false
	return nil
}

func (f *Forward) Activate() error {
	f.l.Lock()
	defer f.l.Unlock()
	if f.Active {
		return errors.New("port forward is already active")
	}
	f.Active = true

	if strings.Contains(f.Fwd.Host.Type, "tcp") {
		l, err := net.Listen("tcp", f.Fwd.Host.String())
		if err != nil {
			f.logger.Error("failed to setup host listener", "type", "tcp", "host", f.Fwd.Host, "error", err)
			return err
		}

		go func() {
			<-f.Ctx.Done()
			l.Close()
		}()

		f.logger.Debug("activated port forward", "type", "tcp", "fwd", f)

		go func() {
			for {
				conn, err := l.Accept()
				if err != nil {
					f.logger.Error("failed to accept incoming connection", "type", "tcp", "fwd", f, "error", err)
					f.cancel()
					return
				}

				target, err := net.Dial("tcp", f.Fwd.Guest.String())
				if err != nil {
					f.logger.Warn("failed to connect to guest", "type", "tcp", "guest", f.Fwd.Guest)
					continue
				}

				ctx, completed := context.WithCancel(f.Ctx)
				f.logger.Debug("initializing new connection stream", "type", "tcp", "fwd", f, "source", conn.RemoteAddr())
				go f.stream(conn, target, completed, "tcp", "outgoing")
				go f.stream(target, conn, completed, "tcp", "incoming")

				go func() {
					select {
					case <-ctx.Done():
					case <-f.Ctx.Done():
					}
					conn.Close()
					target.Close()
				}()
			}
		}()
	}

	if strings.Contains(f.Fwd.Host.Type, "udp") {
		addr := &net.UDPAddr{
			IP:   net.ParseIP(f.Fwd.Host.Host),
			Port: f.Fwd.Host.Port,
		}
		conn, err := net.ListenUDP("udp", addr)
		if err != nil {
			f.logger.Error("failed to setup host listener", "type", "udp", "host", f.Fwd.Host, "error", err)
			return err
		}

		target, err := net.Dial("udp", f.Fwd.Guest.String())
		if err != nil {
			f.logger.Error("failed to connect to guest", "type", "udp", "guest", f.Fwd.Guest, "error", err)
			conn.Close()
			return err
		}

		ctx, completed := context.WithCancel(f.Ctx)
		go func() {
			select {
			case <-ctx.Done():
			case <-f.Ctx.Done():
			}
			conn.Close()
		}()

		f.logger.Debug("initializing connection stream", "type", "udp", "fwd", f)
		go f.stream(conn, target, completed, "udp", "incoming")

		f.logger.Debug("activated port forward", "type", "udp", "fwd", f)
	}

	return nil
}

func (f *Forward) stream(incoming io.ReadCloser, outgoing io.WriteCloser, complete context.CancelFunc, kind, direction string) {
	defer incoming.Close()
	defer outgoing.Close()

	n, err := io.Copy(outgoing, incoming)
	f.logger.Debug("connection stream complete", "direction", direction, "type", kind, "fwd", f, "bytes", n, "error", err)
	complete()
}

type PortForwarding struct {
	forwards []*Forward

	ctx    context.Context
	l      sync.Mutex
	logger hclog.Logger
	s      *settings.PortForwarding
}

func NewPortForwarding(s *settings.Settings, logger hclog.Logger) (*PortForwarding, error) {
	logger = logger.Named("pfwd-service")
	return &PortForwarding{
		forwards: []*Forward{},
		ctx:      context.Background(),
		logger:   logger,
		s:        s.PortForwarding,
	}, nil
}

// Loads all known forwards from the
// persisted settings
func (p *PortForwarding) Load() error {
	p.l.Lock()
	defer p.l.Unlock()

	p.logger.Debug("loading any persisted port forwards")

	for _, f := range p.s.Forwards {
		p.logger.Trace("persisted port forward found", "fwd", f)
		ctx, cancel := context.WithCancel(p.ctx)
		fwd := &Forward{
			Active: false,
			Ctx:    ctx,
			Fwd:    f,
			cancel: cancel,
			logger: p.logger.Named("fwd"),
		}
		p.forwards = append(p.forwards, fwd)
	}
	return nil
}

func (p *PortForwarding) Start() error {
	p.l.Lock()
	defer p.l.Unlock()

	p.logger.Debug("starting port forwarding service")

	for _, f := range p.forwards {
		p.logger.Trace("processing port forward", "fwd", f)
		if f.Active {
			p.logger.Trace("port forward marked as active", "fwd", f)
			continue
		}
		if err := f.Activate(); err != nil {
			return err
		}
	}
	return nil
}

func (p *PortForwarding) Stop() error {
	p.l.Lock()
	defer p.l.Unlock()

	for _, f := range p.forwards {
		if !f.Active {
			continue
		}
		if err := f.Deactivate(); err != nil {
			return err
		}
	}
	return nil
}

func (p *PortForwarding) Add(fwd *settings.Forward) error {
	p.l.Lock()
	defer p.l.Unlock()

	p.logger.Debug("adding new port forward", "fwd", fwd)

	err := p.s.Add(fwd)
	if err != nil {
		p.logger.Error("failed to add port forward", "fwd", fwd, "error", err)
		return err
	}

	ctx, cancel := context.WithCancel(p.ctx)

	f := &Forward{
		Active: false,
		Ctx:    ctx,
		Fwd:    fwd,
		cancel: cancel,
		logger: p.logger.Named("fwd"),
	}

	p.logger.Trace("activating new port forward", "fwd", fwd)
	err = f.Activate()
	if err != nil {
		p.logger.Error("failed to activate new port forward", "fwd", fwd, "error", err)
		return err
	}

	p.forwards = append(p.forwards, f)

	return nil
}

func (p *PortForwarding) Remove(fwd *settings.Forward) error {
	p.l.Lock()
	defer p.l.Unlock()

	p.logger.Debug("removing port forward", "fwd", fwd)

	for i, f := range p.forwards {
		if f.Fwd.Equal(fwd) {
			p.logger.Trace("port forward found for removal", "fwd", fwd)
			p.s.Delete(fwd)
			p.forwards = append(p.forwards[0:i], p.forwards[i+1:]...)
			p.logger.Trace("deactivating port forward", "fwd", fwd)
			err := f.Deactivate()
			if err != nil {
				p.logger.Error("failed to deactivate port forward", "fwd", fwd, "error", err)
				return err
			}
			return nil
		}
	}
	p.logger.Warn("failed to locate port forward for removal", "fwd", fwd)
	return nil
}

func (p *PortForwarding) Fwds() []*Forward {
	return p.forwards
}
