/*
 * SPDX-License-Identifier: Apache-2.0
 * Copyright 2026 Hygon Information Technology Co., Ltd.
 */

// Package log is the single logging entry point for the device plugin.
// It is backed by klog/v2, which is what client-go and apimachinery already use.
package log

import (
	"flag"
	"fmt"
	"strconv"
	"strings"

	"k8s.io/klog/v2"
)

// Severity gates output at the call site.
type Severity int32

const (
	SeverityInfo Severity = iota
	SeverityWarning
	SeverityError
)

var gate = SeverityInfo

// ParseSeverity maps a user-supplied name to a Severity.
func ParseSeverity(name string) (Severity, error) {
	switch strings.ToUpper(strings.TrimSpace(name)) {
	case "INFO":
		return SeverityInfo, nil
	case "WARNING", "WARN":
		return SeverityWarning, nil
	case "ERROR":
		return SeverityError, nil
	default:
		return SeverityInfo, fmt.Errorf("invalid log severity %q, must be one of [INFO | WARNING | ERROR]", name)
	}
}

// Init configures logging for the whole process.
//
// klog gets its own FlagSet on purpose: glog registers v/logtostderr/stderrthreshold/
// alsologtostderr on flag.CommandLine from its init(), and glog is pulled in
// transitively by device-plugin-manager. Sharing the FlagSet would panic with
// "flag redefined". The glog flags are then set to mirror klog so that log output
// from that dependency stays consistent with ours.
//
// Everything goes to stderr; klog's own stderrthreshold is bypassed whenever
// logtostderr is set, so severity gating is enforced by this package instead.
func Init(level int, severity string) error {
	sev, err := ParseSeverity(severity)
	if err != nil {
		return err
	}
	gate = sev

	klogFlags := flag.NewFlagSet("klog", flag.ContinueOnError)
	klog.InitFlags(klogFlags)
	if err := klogFlags.Set("logtostderr", "true"); err != nil {
		return err
	}
	if err := klogFlags.Set("v", strconv.Itoa(level)); err != nil {
		return err
	}

	// Verbose output from the dependency is only meaningful when Info is allowed.
	glogLevel := level
	if gate > SeverityInfo {
		glogLevel = 0
	}
	setGlogFlag("logtostderr", "true")
	setGlogFlag("v", strconv.Itoa(glogLevel))

	return nil
}

// setGlogFlag is best-effort: the flag only exists while glog is linked in.
func setGlogFlag(name, value string) {
	if f := flag.CommandLine.Lookup(name); f != nil {
		_ = f.Value.Set(value)
	}
}

func Flush() {
	klog.Flush()
}

// Depth is 1 so that the reported file:line is the caller, not this wrapper.

func Info(args ...interface{}) {
	if gate > SeverityInfo {
		return
	}
	klog.InfoDepth(1, args...)
}

func Infof(format string, args ...interface{}) {
	if gate > SeverityInfo {
		return
	}
	klog.InfofDepth(1, format, args...)
}

func Warningf(format string, args ...interface{}) {
	if gate > SeverityWarning {
		return
	}
	klog.WarningfDepth(1, format, args...)
}

func Error(args ...interface{}) {
	klog.ErrorDepth(1, args...)
}

func Errorf(format string, args ...interface{}) {
	klog.ErrorfDepth(1, format, args...)
}

func Errorln(args ...interface{}) {
	klog.ErrorlnDepth(1, args...)
}

// Verbose mirrors klog.Verbose so that V(n).Infof reads the same at call sites.
type Verbose struct {
	enabled bool
}

// V reports whether logging at the given verbosity level is enabled.
func V(level klog.Level) Verbose {
	if gate > SeverityInfo {
		return Verbose{enabled: false}
	}
	return Verbose{enabled: klog.V(level).Enabled()}
}

func (v Verbose) Enabled() bool {
	return v.enabled
}

func (v Verbose) Info(args ...interface{}) {
	if !v.enabled {
		return
	}
	klog.InfoDepth(1, args...)
}

func (v Verbose) Infof(format string, args ...interface{}) {
	if !v.enabled {
		return
	}
	klog.InfofDepth(1, format, args...)
}

func (v Verbose) Infoln(args ...interface{}) {
	if !v.enabled {
		return
	}
	klog.InfolnDepth(1, args...)
}
