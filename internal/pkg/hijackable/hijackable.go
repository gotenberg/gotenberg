package hijackable

import "fmt"

type Hijackable interface {
	Hijack() error
}

func HijackOnError(h Hijackable, err error) error {
	if hijackError := h.Hijack(); hijackError != nil {
		return fmt.Errorf("error on hijack: %v: root error: %v", hijackError, err)
	}
	return err
}
