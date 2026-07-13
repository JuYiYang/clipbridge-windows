//go:build !windows

package singleinstance

import "errors"

var ErrAlreadyRunning = errors.New("clipbridge is already running")

type Lock struct{}

func Acquire() (*Lock, error) {
	return &Lock{}, nil
}

func (l *Lock) Release() {}
