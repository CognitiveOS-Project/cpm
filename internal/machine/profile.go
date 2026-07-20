package machine

import (
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

type Profile struct {
	Hardware HardwareProfile `json:"hardware"`
	Software SoftwareProfile `json:"software"`
	Network  NetworkProfile  `json:"network"`
}

type HardwareProfile struct {
	CPU       string `json:"cpu,omitempty"`
	Cores     int    `json:"cores,omitempty"`
	Arch      string `json:"arch,omitempty"`
	RAMMB     int    `json:"ram_mb,omitempty"`
	StorageMB int    `json:"storage_mb,omitempty"`
	GPU       string `json:"gpu,omitempty"`
	TPM       bool   `json:"tpm,omitempty"`
	MachineID string `json:"machine_id,omitempty"`
}

type SoftwareProfile struct {
	OS         string `json:"os,omitempty"`
	Kernel     string `json:"kernel,omitempty"`
	Distro     string `json:"distro,omitempty"`
	CPMVersion string `json:"cpm_version,omitempty"`
	Packages   int    `json:"packages,omitempty"`
	Services   int    `json:"services,omitempty"`
}

type NetworkProfile struct {
	IP string `json:"ip,omitempty"`
}

func Gather() *Profile {
	p := &Profile{}
	p.Hardware = gatherHardware()
	p.Software = gatherSoftware()
	p.Network = gatherNetwork()
	return p
}

func gatherHardware() HardwareProfile {
	h := HardwareProfile{
		Cores: runtime.NumCPU(),
		Arch:  runtime.GOARCH,
	}

	h.CPU = readFirstLine("/proc/cpuinfo")
	h.RAMMB = readMemInfo()
	h.MachineID = readMachineID()
	h.TPM = fileExists("/dev/tpm0") || fileExists("/dev/tpmrm0")
	h.GPU = detectGPU()
	h.StorageMB = 0

	return h
}

func gatherSoftware() SoftwareProfile {
	s := SoftwareProfile{
		OS:     runtime.GOOS,
		Kernel: runCommand("uname", "-r"),
		Distro: readDistro(),
	}

	s.Packages = countPackages()
	s.Services = countServices()

	return s
}

func gatherNetwork() NetworkProfile {
	n := NetworkProfile{
		IP: getLocalIP(),
	}
	return n
}

func readFirstLine(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "model name") || strings.HasPrefix(line, "Hardware") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return ""
}

func readMemInfo() int {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "MemTotal:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				kb, _ := strconv.Atoi(parts[1])
				return kb / 1024
			}
		}
	}
	return 0
}

func readMachineID() string {
	for _, path := range []string{"/etc/machine-id", "/var/lib/dbus/machine-id"} {
		data, err := os.ReadFile(path)
		if err == nil {
			id := strings.TrimSpace(string(data))
			if id != "" {
				return id
			}
		}
	}
	return ""
}

func readDistro() string {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "PRETTY_NAME=") {
			return strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), "\"")
		}
	}
	return ""
}

func detectGPU() string {
	out := runCommand("lspci", "-nn")
	for _, line := range strings.Split(out, "\n") {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "vga") || strings.Contains(lower, "3d") || strings.Contains(lower, "display") {
			return strings.TrimSpace(line)
		}
	}
	return ""
}

func countPackages() int {
	count := 0

	if _, err := os.Stat("/usr/bin/dpkg"); err == nil {
		out := runCommand("dpkg", "-l")
		for _, line := range strings.Split(out, "\n") {
			if strings.HasPrefix(line, "ii") {
				count++
			}
		}
		return count
	}

	if _, err := os.Stat("/sbin/apk"); err == nil {
		out := runCommand("apk", "list", "--installed")
		count = len(strings.Split(strings.TrimSpace(out), "\n"))
		if count == 1 && strings.TrimSpace(out) == "" {
			count = 0
		}
		return count
	}

	return count
}

func countServices() int {
	out := runCommand("ls", "/etc/init.d/")
	if out == "" {
		return 0
	}
	return len(strings.Split(strings.TrimSpace(out), "\n"))
}

func getLocalIP() string {
	out := runCommand("hostname", "-I")
	if out == "" {
		return ""
	}
	parts := strings.Fields(out)
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func runCommand(name string, args ...string) string {
	cmd := exec.Command(name, args...)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
