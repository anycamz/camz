// Copyright © 2023 Sloan Childers
package sink

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog/log"
)

type ShutdownHandler struct {
	listeners []func()
}

func NewShutdownHandler() *ShutdownHandler {
	return &ShutdownHandler{}
}

func (x *ShutdownHandler) AddListener(f func()) {
	x.listeners = append(x.listeners, f)
}

func (x *ShutdownHandler) Listen() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		s := <-c
		log.Info().Str("component", "shutdown").Str("signal", s.String()).Msg("signal nofify")
		for _, ShutItDown := range x.listeners {
			ShutItDown()
		}
		os.Exit(0)
	}()
}
