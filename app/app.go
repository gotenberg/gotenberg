package app

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gulien/gotenberg/app/config"
	"github.com/gulien/gotenberg/app/handlers"
	"github.com/gulien/gotenberg/app/handlers/converter/process"
	"github.com/gulien/gotenberg/app/logger"

	"github.com/gorilla/mux"
)

type App struct {
	version string
	config  *config.AppConfig
	Server  *http.Server
}

func NewApp(version string) (*App, error) {
	c, err := config.NewAppConfig()
	if err != nil {
		return nil, err
	}

	a := &App{}
	a.version = version
	a.config = c

	// defines our application logging.
	logger.SetLevel(a.config.Logs.Level)
	logger.SetFormatter(a.config.Logs.Formatter)

	// defines our application router.
	r := mux.NewRouter()
	r.Handle("/", handlers.GetHandlersChain())

	a.Server = &http.Server{
		Addr: fmt.Sprintf(":%s", a.config.Port),
		// good practice to set timeouts to avoid Slowloris attacks.
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      r,
	}

	return a, nil
}

func (a *App) Run() error {
	process.Load(a.config.CommandsConfig)
	logger.Infof("Starting Gotenberg version %s", a.version)
	logger.Infof("Listening on port %s", a.config.Port)

	return a.Server.ListenAndServe()
}
