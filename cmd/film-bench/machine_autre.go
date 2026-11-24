//go:build !windows

package main

import (
	"os"
	"strconv"
	"strings"
)

// processeur lit le nom du processeur dans /proc/cpuinfo (vide si absent).
func processeur() string {
	brut, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return ""
	}
	for l := range strings.SplitSeq(string(brut), "\n") {
		if k, v, ok := strings.Cut(l, ":"); ok && strings.TrimSpace(k) == "model name" {
			return strings.Join(strings.Fields(v), " ")
		}
	}
	return ""
}

// memoireMio lit la memoire totale dans /proc/meminfo, en Mio (0 si absent).
func memoireMio() uint64 {
	brut, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0
	}
	for l := range strings.SplitSeq(string(brut), "\n") {
		champs := strings.Fields(l)
		if len(champs) >= 2 && champs[0] == "MemTotal:" {
			if ko, err := strconv.ParseUint(champs[1], 10, 64); err == nil {
				return ko >> 10
			}
		}
	}
	return 0
}
