package pm2

type dummyProcess struct {
	isViable bool
}

// NewDummyProcess returns a dummy
// process.
func NewDummyProcess(isViable bool) Process {
	return dummyProcess{
		isViable: isViable,
	}
}

func (p dummyProcess) Fullname() string {
	return "dummy process"
}

func (p dummyProcess) Start() error {
	return nil
}

func (p dummyProcess) IsViable() bool {
	return p.isViable
}

func (p dummyProcess) Stop() error {
	return nil
}

func (p dummyProcess) args() []string {
	return []string{}
}

func (p dummyProcess) binary() string {
	return "dummy"
}

func (p dummyProcess) warmup() {
	// let's do nothing.
}

// Compile-time checks to ensure type implements desired interfaces.
var (
	_ = Process(new(dummyProcess))
)
