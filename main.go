// Copyright © 2023 Sloan Childers
package main

import (
	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
	"github.com/osintami/camz/axis"
	"github.com/osintami/camz/base"
	"github.com/osintami/camz/blackjack"
	"github.com/osintami/camz/opencv"
	"github.com/osintami/camz/sink"
	"github.com/rs/zerolog/log"
)

func main() {

	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal().Str("component", "server").Msg("load .env")
		return
	}
	serverCfg := &Config{}
	sink.LoadEnv(serverCfg)
	sink.InitLogger(serverCfg.LogLevel)
	//sink.PrintEnvironment()

	shutdown := sink.NewShutdownHandler()

	log.Info().Msg("It's alive!")

	config := &base.CameraConfig{}
	err = sink.LoadJson("./camera.json", config)
	if err != nil {
		log.Fatal().Str("component", "server").Msg("load camera.json")
		return
	}

	motion := opencv.NewMotion(config)
	const ONE_SECOND = 1
	gps := base.NewGPS(ONE_SECOND)

	var webcam base.IDriver

	switch config.Plugin {
	case "opencv":
		webcam = opencv.NewDriver(config)
	case "blackjack":
		webcam = blackjack.NewDriver(config)
	case "axis241q":
		webcam = axis.NewDriver(config)
	}
	err = webcam.Open()
	if err != nil {
		log.Fatal().Err(err).Msg("webcam")
		return
	}
	webcam.Stream()

	shutdown.AddListener(webcam.Stop)

	handlers := NewCamzServer(webcam, motion, gps, config)
	router := chi.NewMux()
	router.Route(serverCfg.PathPrefix, func(r chi.Router) {
		r.Get("/v1/stream", handlers.StreamHandler)
		// change/view settings
		r.Post("/v1/config", handlers.ConfigUpdateHandler)
		r.Get("/v1/config", handlers.ConfigReadHandler)
		// list formats and frame sizes supported by device
		r.Get("/v1/formats", handlers.FormatsHandler)
		r.Get("/v1/command", handlers.CommandHandler)
	})

	shutdown.Listen()

	err = sink.ListenAndServe(serverCfg.ListenAddr, "", "", router)
	if err != nil {
		log.Error().Err(err).Str("component", "server").Msg("listen and serve")
	}
}
