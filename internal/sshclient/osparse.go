package sshclient

import (
	"fmt"
	"strconv"
	"strings"
)

func ParseUptimeSeconds(out string) (float64, error) {
	// /proc/uptime: "<uptime> <idle>"
	fields := strings.Fields(out)
	if len(fields) < 1 {
		return 0, fmt.Errorf("bad uptime: %q", out)
	}
	return strconv.ParseFloat(fields[0], 64)
}

func ParseLoad1(out string) (float64, error) {
	// /proc/loadavg: "0.10 0.20 0.30 1/123 4567"
	fields := strings.Fields(out)
	if len(fields) < 1 {
		return 0, fmt.Errorf("bad loadavg: %q", out)
	}
	return strconv.ParseFloat(fields[0], 64)
}

func ParseMeminfo(out string) (totalBytes, availBytes float64, err error) {
	// ต้องการ MemTotal + MemAvailable (kB)
	var totalKB, availKB float64

	lines := strings.Split(out, "\n")
	for _, ln := range lines {
		ln = strings.TrimSpace(ln)
		if strings.HasPrefix(ln, "MemTotal:") {
			totalKB, _ = parseMeminfoKB(ln)
		}
		if strings.HasPrefix(ln, "MemAvailable:") {
			availKB, _ = parseMeminfoKB(ln)
		}
	}

	if totalKB <= 0 || availKB < 0 {
		return 0, 0, fmt.Errorf("missing MemTotal/MemAvailable")
	}
	return totalKB * 1024, availKB * 1024, nil
}

func parseMeminfoKB(line string) (float64, error) {
	// "MemTotal:       4015356 kB"
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return 0, fmt.Errorf("bad meminfo line: %q", line)
	}
	return strconv.ParseFloat(fields[1], 64)
}
