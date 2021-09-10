package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/xxf098/lite-proxy/config"
	_ "github.com/xxf098/lite-proxy/config"
	"github.com/xxf098/lite-proxy/dns"
	"github.com/xxf098/lite-proxy/outbound"
	"github.com/xxf098/lite-proxy/proxy"
	"github.com/xxf098/lite-proxy/request"
	"github.com/xxf098/lite-proxy/transport/resolver"
	"github.com/xxf098/lite-proxy/tunnel"
	"github.com/xxf098/lite-proxy/tunnel/adapter"
	"github.com/xxf098/lite-proxy/tunnel/http"
	"github.com/xxf098/lite-proxy/tunnel/socks"
	"github.com/xxf098/lite-proxy/utils"
)

func startInstance(c Config) (*proxy.Proxy, error) {
	ctx, cancel := context.WithCancel(context.Background())
	ctx = context.WithValue(ctx, "LocalHost", c.LocalHost)
	ctx = context.WithValue(ctx, "LocalPort", c.LocalPort)
	adapterServer, err := adapter.NewServer(ctx, nil)
	if err != nil {
		return nil, err
	}
	httpServer, err := http.NewServer(ctx, adapterServer)
	if err != nil {
		return nil, err
	}
	socksServer, err := socks.NewServer(ctx, adapterServer)
	if err != nil {
		return nil, err
	}
	sources := []tunnel.Server{httpServer, socksServer}
	sink, err := createSink(ctx, c.Link)
	if err != nil {
		return nil, err
	}
	go func(link string) {
		if cfg, err := config.Link2Config(c.Link); err == nil {
			opt := request.PingOption{
				Attempts: 2,
				TimeOut:  1200 * time.Millisecond,
			}
			info := fmt.Sprintf("%s %s:%d", cfg.Remarks, cfg.Server, cfg.Port)
			if elapse, err := request.PingLinkInternal(link, opt); err == nil {
				info = fmt.Sprintf("%s \033[32m%dms\033[0m", info, elapse)
			} else {
				info = fmt.Sprintf("\033[32m%sms\033[0m", err.Error())
			}
			log.Print(info)
		}
	}(c.Link)
	setDefaultResolver()
	p := proxy.NewProxy(ctx, cancel, sources, sink)
	return p, nil
}

func createSink(ctx context.Context, link string) (tunnel.Client, error) {
	var d outbound.Dialer
	matches, err := utils.CheckLink(link)
	if err != nil {
		return nil, err
	}
	creator, err := outbound.GetDialerCreator(matches[1])
	if err != nil {
		return nil, err
	}
	d, err = creator(link)
	if err != nil {
		return nil, err
	}
	if d != nil {
		return proxy.NewClient(ctx, d), nil
	}

	return nil, errors.New("not supported link")
}

func setDefaultResolver() {
	resolver.DefaultResolver = dns.DefaultResolver()
}
