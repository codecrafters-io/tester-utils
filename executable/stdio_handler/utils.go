package stdio_handler

import (
	"errors"
	"io"
	"os"
	"reflect"
)

// closeIfNotNil closes an io.Closer if it is not already closed
func closeIfNotNil(c io.Closer) error {
	v := reflect.ValueOf(c)

	if v.Kind() == reflect.Pointer && v.IsNil() {
		return nil
	}

	return c.Close()
}

// closeIfOpen closes an io.Closer if it is not already closed
func closeIfOpen(c io.Closer) error {
	err := c.Close()

	if err != nil && !errors.Is(err, os.ErrClosed) {
		return err
	}

	return nil
}

// closeAllWithCloserFunc makes best effort (attempts to close all even in case of error)
// to close all the io.Closer interfacs using the provided closer function.
func closeAllWithCloserFunc(closer func(io.Closer) error, streams ...io.Closer) error {
	var firstError error
	for _, stream := range streams {
		if err := closer(stream); err != nil && firstError == nil {
			firstError = err
		}
	}
	return firstError
}
