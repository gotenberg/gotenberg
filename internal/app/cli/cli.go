package cli

import (
	"os"

	"github.com/thecodingmachine/gotenberg/internal/app/cli/api"
	"github.com/thecodingmachine/gotenberg/internal/pkg/docker"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	app    = kingpin.New("gotenberg", "A command-line tool for generating PDF from various sources.")
	server = app.Command("server", "Start a stateless API on port 3000.")
)

func SetVersion(version string) { kingpin.Version(version) }

func Run() error {
	switch kingpin.MustParse(app.Parse(os.Args[1:])) {
	case server.FullCommand():
		if err := docker.StartChromeHeadless(); err != nil {
			return err
		}
		if err := docker.StartOfficeHeadless(); err != nil {
			return err
		}
		return api.Start()
	}
	return nil
}
