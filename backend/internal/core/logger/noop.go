package logger

import (
	"context"

	"github.com/itsLeonB/ezutil/v2"
)

// noop is Global's default until Init runs, so logging before (or without)
// Init, e.g. in tests, is a safe no-op instead of a nil-pointer panic.
type noop struct{}

func (noop) Debug(...any)                                {}
func (noop) Info(...any)                                 {}
func (noop) Warn(...any)                                 {}
func (noop) Error(...any)                                {}
func (noop) Fatal(...any)                                {}
func (noop) Debugf(string, ...any)                       {}
func (noop) Infof(string, ...any)                        {}
func (noop) Warnf(string, ...any)                        {}
func (noop) Errorf(string, ...any)                       {}
func (noop) Fatalf(string, ...any)                       {}
func (noop) Printf(string, ...any)                       {}
func (n noop) WithError(error) ezutil.Logger             { return n }
func (n noop) WithField(string, any) ezutil.Logger       { return n }
func (n noop) WithFields(map[string]any) ezutil.Logger   { return n }
func (n noop) WithContext(context.Context) ezutil.Logger { return n }
