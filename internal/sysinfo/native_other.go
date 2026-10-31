//go:build !windows && !darwin && !dragonfly && !freebsd && !netbsd && !openbsd

package sysinfo

func native() Collector { return Unsupported{} }
