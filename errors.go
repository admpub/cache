package cache

import "errors"

var (
	ErrNotFound     = errors.New("not found")
	ErrExpired      = errors.New("expired")
	ErrNotSupported = errors.New("not supported operation")
)

// IsDataStatusError reports whether the error is either ErrNotFound or ErrExpired.
func IsDataStatusError(err error) bool {
	return errors.Is(err, ErrNotFound) || errors.Is(err, ErrExpired)
}

// IsNotFound reports whether an error is ErrNotFound.
func IsNotFound(err error) bool { return errors.Is(err, ErrNotFound) }

// IsExpired reports whether the given error indicates an expired cache entry.
func IsExpired(err error) bool { return errors.Is(err, ErrExpired) }

// IsNotSupported reports whether an error indicates the operation is not supported.
func IsNotSupported(err error) bool { return errors.Is(err, ErrNotSupported) }
