package pm2

// Unoconv facilitates starting or shutting down
// unoconv listener with PM2.
type Unoconv struct {
	heuristicState int32
}

// Start starts unoconv listener with PM2.
func (u *Unoconv) Start() error {
	return startProcess(u)
}

// Shutdown stops unoconv listener and
// removes it from the list of PM2
// processes.
func (u *Unoconv) Shutdown() error {
	return shutdownProcess(u)
}

// State returns the current state of
// unoconv listener process.
func (u *Unoconv) State() int32 {
	return u.heuristicState
}

func (u *Unoconv) state(state int32) {
	u.heuristicState = state
}

func (u *Unoconv) args() []string {
	return []string{
		"--listener",
		"--verbose",
	}
}

func (u *Unoconv) name() string {
	return "unoconv"
}

func (u *Unoconv) fullname() string {
	return "unoconv listener"
}

func (u *Unoconv) viable() bool {
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
