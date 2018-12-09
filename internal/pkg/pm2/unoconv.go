package pm2

// Unoconv facilitates starting or shutting down
// unoconv listener with PM2.
type Unoconv struct{}

// Launch starts unoconv listener with PM2.
func (u *Unoconv) Launch() error {
	return launch(u)
}

// Shutdown stops unoconv listener and
// removes it from the list of PM2
// processes.
func (u *Unoconv) Shutdown() error {
	return shutdown(u)
}

func (u *Unoconv) getArgs() []string {
	return []string{
		"--listener",
		"--verbose",
	}
}

func (u *Unoconv) getName() string {
	return "unoconv"
}

func (u *Unoconv) getFullname() string {
	return "unoconv listener"
}

func (u *Unoconv) isViable() bool {
	// TODO find a way to check if
	// unoconv is correctly started?
	return true
}

func (u *Unoconv) warmup() {
	// let's do nothing.
}

// Compile-time checks to ensure type implements desired interfaces.
var (
	_ = Process(new(Unoconv))
)
