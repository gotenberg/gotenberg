package printer

// Printer is a type that can create a PDF file from a source.
// The source is defined in the underlying implementation.
type Printer interface {
	Print(destination string) error
}
