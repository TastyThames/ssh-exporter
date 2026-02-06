package sshclient

import "fmt"

// AllowedCommand = คำสั่งที่อนุญาตให้รันเท่านั้น
type AllowedCommand struct {
	kind string
}

func (c AllowedCommand) String() string {
	switch c.kind {
	case "meminfo":
		return "cat /proc/meminfo"
	case "loadavg":
		return "cat /proc/loadavg"
	case "uptime":
		return "cat /proc/uptime"
	case "netdev":
		return "cat /proc/net/dev"
	default:
		return "false"
	}
}

func CmdMeminfo() AllowedCommand { return AllowedCommand{kind: "meminfo"} }
func CmdLoadavg() AllowedCommand { return AllowedCommand{kind: "loadavg"} }
func CmdUptime() AllowedCommand  { return AllowedCommand{kind: "uptime"} }
func CmdNetDev() AllowedCommand  { return AllowedCommand{kind: "netdev"} }

func ErrUnsupported(cmd AllowedCommand) error {
	return fmt.Errorf("unsupported command kind=%q", cmd.kind)
}
