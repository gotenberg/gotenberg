package pm2

type unoconv struct {
	manager *processManager
}

// NewUnoconv retruns a unoconv listener
// process.
func NewUnoconv(debug bool) Process {
	return &unoconv{
		manager: &processManager{verbose: debug},
	}
}

func (p *unoconv) Fullname() string {
	return "unoconv listener"
}

func (p *unoconv) Start() error {
	return p.manager.start(p)
}

func (p *unoconv) Shutdown() error {
	return p.manager.shutdown(p)
}

func (p *unoconv) args() []string {
	return []string{
		"--listener",
		"--verbose",
	}
}

func (p *unoconv) name() string {
	return "unoconv"
}

func (p *unoconv) viable() bool {
	// TODO find a way to check if
	// the unoconv listener
	// is correctly started?
	return true
}

func (p *unoconv) warmup() {
	// let's do nothing.
}

// Compile-time checks to ensure type implements desired interfaces.
var (
	_ = Process(new(unoconv))
)
