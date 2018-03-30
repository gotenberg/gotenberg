// Package app is the entry point of the application.
package app

import (
	"fmt"
	"net/http"

	"github.com/gulien/gotenberg/app/config"
	"github.com/gulien/gotenberg/app/handlers"
	"github.com/gulien/gotenberg/app/handlers/converter/process"
	"github.com/gulien/gotenberg/app/logger"

	"github.com/gorilla/mux"
)

// App gathers all data required to start the application.
type App struct {
	// version is the application version as defined in the main package.
	version string
	// config is the application configuration.
	// It's populated thanks to the gotenberg.yml file.
	config *config.AppConfig
	// Server is the instance of http.Server used by the application.
	Server *http.Server
}

// NewApp instantiates the application by loading the configuration from the
// gotenberg.yml file.
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
		Addr:    fmt.Sprintf(":%s", a.config.Port),
		Handler: r,
	}

	return a, nil
}

// Run starts the server.
func (a *App) Run() error {
	process.Load(a.config.CommandsConfig)
	logger.Infof("Starting Gotenberg version %s", a.version)
	logger.Infof("Listening on port %s", a.config.Port)

	return a.Server.ListenAndServe()
}
