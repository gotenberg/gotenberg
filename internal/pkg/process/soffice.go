package process

import (
	"fmt"
	"time"

	"github.com/thecodingmachine/gotenberg/internal/pkg/xlog"
)

const SofficeKey Key = "soffice"

type sofficeProcess struct {
	id   string
	host string
	port int
}

// NewSofficeProcess returns a LibreOffice
// headless process.
func NewSofficeProcess(id, host string, port int) Process {
	return sofficeProcess{
		id:   id,
		host: host,
		port: port,
	}
}

func (p sofficeProcess) ID() string {
	return p.id
}

func (p sofficeProcess) Host() string {
	return p.host
}

func (p sofficeProcess) Port() int {
	return p.port
}

func (p sofficeProcess) binary() string {
	return "soffice"
}

func (p sofficeProcess) args() []string {
	return []string{
		// see https://ask.libreoffice.org/en/question/42975/how-can-i-run-multiple-instances-of-sofficebin-at-a-time/.
		fmt.Sprintf("-env:UserInstallation=file:///tmp/%d", p.port),
		"--headless",
		"--invisible",
		"--nocrashreport",
		"--nodefault",
		"--nofirststartwizard",
		"--nologo",
		"--norestore",
		fmt.Sprintf("--accept=socket,host=%s,port=%d,tcpNoDelay=1;urp;StarOffice.ComponentContext", p.host, p.port),
	}
}

func (p sofficeProcess) warmupTime() time.Duration {
	return 3 * time.Second
}

func (p sofficeProcess) viabilityFunc() func(logger xlog.Logger) bool {
	const op string = "process.sofficeProcess.viabilityFunc"
	return func(logger xlog.Logger) bool {
		// TODO find a way to check.
		return true
	}
}

// Compile-time checks to ensure type implements desired interfaces.
var (
	_ = Process(new(sofficeProcess))
)
