// Copyright © 2023 Sloan Childers
package sink

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/caarlos0/env/v6"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

func InitLogger(level string) {
	parsedLevel, err := zerolog.ParseLevel(strings.ToLower(level))
	if err != nil {
		log.Fatal().Err(err).Msg("unable to configure logger")
	}
	zerolog.SetGlobalLevel(parsedLevel)
}

func LoadEnv(output interface{}) {

	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal().Msg(".env")
		return
	}

	if err := env.Parse(output); err != nil {
		log.Fatal().Err(err).Msg("environment")
	}
	// PrintEnvironment()
}

func LoadJson(fileName string, cfg interface{}) error {
	// load collector configuration
	fh, err := os.Open(fileName)
	if err != nil {
		log.Error().Err(err).Str("component", "utils").Str("file", fileName).Msg("load json open")
		return err
	}
	defer fh.Close()

	obj, err := io.ReadAll(fh)
	if err != nil {
		log.Error().Err(err).Str("component", "utils").Str("file", fileName).Msg("load json read")
		return err
	}
	err = json.Unmarshal(obj, cfg)
	if err != nil {
		log.Error().Err(err).Str("component", "utils").Str("file", fileName).Msg("load json parse")
		return err
	}
	return nil
}

func ListenAndServe(ListenAddr, SSLCertFile, SSLKeyFile string, router http.Handler) error {
	server := &http.Server{Addr: ListenAddr, Handler: router}
	var err error
	if SSLCertFile != "" {
		err = server.ListenAndServeTLS(SSLCertFile, SSLKeyFile)
	} else {
		err = server.ListenAndServe()
	}
	return err
}

func PrintEnvironment() {
	env := os.Environ()
	for _, variable := range env {
		pair := strings.Split(variable, "=")
		value := pair[1]
		if strings.Contains(pair[0], "API_KEY") || strings.Contains(pair[0], "PASS") || strings.Contains(pair[0], "USER") {
			value = "<masked>"
		}
		log.Info().Str(pair[0], value).Msg("environment")
	}
}

func Param(r *http.Request, key string) string {
	return chi.URLParam(r, key)
}

func SendError(w http.ResponseWriter, err error, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": err.Error(),
	})
}

func SendPrettyJSON(ctx context.Context, w http.ResponseWriter, data interface{}) {
	span, _ := tracer.StartSpanFromContext(ctx, "rendering_json")
	defer span.Finish()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "    ")
	err := encoder.Encode(data)
	if err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Interface("data", data).Msg("unable to pretty json")
	}
}
