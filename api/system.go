package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/ksauraj/ksau-oned-api/azure"
	"github.com/ksauraj/ksau-oned-api/config"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
)

// Basic system info for the /system endpoint
type SystemInfo struct {
	Status string `json:"status"`
	Data   struct {
		CPU struct {
			Model    string   `json:"model"`
			Cores    int      `json:"cores"`
			Usage    float64  `json:"usage"`
			LoadPerc []string `json:"load_percentage"`
		} `json:"cpu"`
		Memory struct {
			Total       uint64  `json:"total"`
			Used        uint64  `json:"used"`
			Free        uint64  `json:"free"`
			UsedPercent float64 `json:"used_percent"`
		} `json:"memory"`
		System struct {
			Hostname     string    `json:"hostname"`
			OS           string    `json:"os"`
			Platform     string    `json:"platform"`
			Kernel       string    `json:"kernel"`
			Architecture string    `json:"architecture"`
			ServerTime   time.Time `json:"server_time"`
			Uptime       uint64    `json:"uptime"`
		} `json:"system"`
	} `json:"data"`
}

// Neofetch-style info structure
type NeofetchInfo struct {
	ASCIIArt string `json:"ascii_art"`
	Colors   Colors `json:"colors"`
	System   struct {
		User        string    `json:"user"`
		Hostname    string    `json:"hostname"`
		Distro      string    `json:"distro"`
		Kernel      string    `json:"kernel"`
		Uptime      string    `json:"uptime"`
		Shell       string    `json:"shell"`
		CPU         string    `json:"cpu"`
		Memory      string    `json:"memory"`
		DiskUsage   string    `json:"disk_usage"`
		LocalIP     string    `json:"local_ip"`
		ServerTime  time.Time `json:"server_time"`
		LoadAverage []float64 `json:"load_average"`
	} `json:"system"`
	Performance struct {
		CPUUsage     float64   `json:"cpu_usage"`
		MemoryUsage  float64   `json:"memory_usage"`
		CPUFrequency float64   `json:"cpu_frequency"`
		CoreLoads    []float64 `json:"core_loads"`
	} `json:"performance"`
}

type Colors struct {
	Primary   string `json:"primary"`
	Secondary string `json:"secondary"`
	Accent    string `json:"accent"`
}

// RemoteQuota represents quota information for a remote storage
type RemoteQuotaInfo struct {
	Status string                  `json:"status"`
	Data   map[string]*RemoteQuota `json:"data"`
}

type RemoteQuota struct {
	Total     string `json:"total"`
	Used      string `json:"used"`
	Remaining string `json:"remaining"`
	Deleted   string `json:"deleted"`
}

// SystemHandler provides basic system information
func SystemHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	info := SystemInfo{Status: "success"}

	// Get CPU info
	cpuInfo, err := cpu.Info()
	if err == nil && len(cpuInfo) > 0 {
		info.Data.CPU.Model = cpuInfo[0].ModelName
		info.Data.CPU.Cores = runtime.NumCPU()
	}

	// Get CPU usage
	cpuPercent, err := cpu.Percent(0, true)
	if err == nil {
		info.Data.CPU.LoadPerc = make([]string, len(cpuPercent))
		var totalUsage float64
		for i, usage := range cpuPercent {
			info.Data.CPU.LoadPerc[i] = fmt.Sprintf("%.1f%%", usage)
			totalUsage += usage
		}
		if len(cpuPercent) > 0 {
			info.Data.CPU.Usage = totalUsage / float64(len(cpuPercent))
		}
	}

	// Get memory info
	memInfo, err := mem.VirtualMemory()
	if err == nil {
		info.Data.Memory.Total = memInfo.Total
		info.Data.Memory.Used = memInfo.Used
		info.Data.Memory.Free = memInfo.Free
		info.Data.Memory.UsedPercent = memInfo.UsedPercent
	}

	// Get system info
	hostInfo, err := host.Info()
	if err == nil {
		info.Data.System.Hostname = hostInfo.Hostname
		info.Data.System.OS = hostInfo.OS
		info.Data.System.Platform = hostInfo.Platform
		info.Data.System.Kernel = hostInfo.KernelVersion
		info.Data.System.Architecture = runtime.GOARCH
		info.Data.System.Uptime = hostInfo.Uptime
	}

	info.Data.System.ServerTime = time.Now()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

// getASCIIArt returns ASCII art for different OS types
func getASCIIArt(osType string) string {
	return `
    ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⣀⣀⣀⣀⠀⠀⠀⠀⠀⠀
    ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣠⣶⠟⠛⠛⠛⠛⠛⣛⣻⣿⣿⣿⣿⣿⣟⣛⣛⣛⠛⠒⠲⠶⠦⣤⣤⣤⣀⡀⠀⠀
    ⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⣼⠏⠁⠀⠀⢀⣤⠶⣛⠋⠉⠉⠀⠀⠀⠈⠙⢿⠭⠱⠃⠀⠀⠀⠀⠀⠈⠙⠻⠗⠀⠀
    ⠀⠀⠀⠀⠀⠀⠀⠀⢠⡟⠁⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
    ⠀⠀⠀⠀⠀⠀⠀⣠⠏⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠠⣄⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
    ⠀⠀⠀⠀⠀⠀⡰⠃⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠸⣄⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀`
}

func executeCommand(command string, args ...string) string {
	cmd := exec.Command(command, args...)
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error executing command %s: %v", command, err)
		return ""
	}
	return strings.TrimSpace(string(output))
}

func formatUptime(seconds uint64) string {
	days := seconds / (60 * 60 * 24)
	hours := (seconds % (60 * 60 * 24)) / (60 * 60)
	minutes := (seconds % (60 * 60)) / 60

	if days > 0 {
		return fmt.Sprintf("%d days, %d hours, %d minutes", days, hours, minutes)
	} else if hours > 0 {
		return fmt.Sprintf("%d hours, %d minutes", hours, minutes)
	}
	return fmt.Sprintf("%d minutes", minutes)
}

func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// NeofetchHandler provides detailed system information in a neofetch-like format
// QuotaHandler returns quota information for all remotes
func QuotaHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := RemoteQuotaInfo{
		Status: "success",
		Data:   make(map[string]*RemoteQuota),
	}

	// Create HTTP client
	httpClient := &http.Client{Timeout: 30 * time.Second}

	// Get embedded config data
	configData := config.GetRcloneConfig()

	// Get list of remotes from config
	remotes := ParseRemotes(string(configData))

	// Get quota for each remote
	for _, remote := range remotes {
		client, err := azure.NewAzureClientFromRcloneConfigData(configData, remote)
		if err != nil {
			log.Printf("Error creating Azure client for remote %s: %v", remote, err)
			continue
		}

		quota, err := client.GetDriveQuota(httpClient)
		if err != nil {
			log.Printf("Error getting quota for remote %s: %v", remote, err)
			continue
		}

		response.Data[remote] = &RemoteQuota{
			Total:     formatBytes(uint64(quota.Total)),
			Used:      formatBytes(uint64(quota.Used)),
			Remaining: formatBytes(uint64(quota.Remaining)),
			Deleted:   formatBytes(uint64(quota.Deleted)),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ParseRemotes extracts remote names from rclone config
func ParseRemotes(config string) []string {
	var remotes []string
	lines := strings.Split(config, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			remote := strings.Trim(line, "[]")
			if remote != "" {
				remotes = append(remotes, remote)
			}
		}
	}
	return remotes
}

func NeofetchHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	info := NeofetchInfo{}
	hostInfo, _ := host.Info()
	info.System.Hostname = hostInfo.Hostname
	info.System.Distro = fmt.Sprintf("%s %s", hostInfo.Platform, hostInfo.PlatformVersion)
	info.System.Kernel = hostInfo.KernelVersion
	info.System.Uptime = formatUptime(hostInfo.Uptime)
	info.System.Shell = os.Getenv("SHELL")
	info.System.ServerTime = time.Now()

	// Get CPU info
	cpuInfo, _ := cpu.Info()
	if len(cpuInfo) > 0 {
		info.System.CPU = fmt.Sprintf("%s (%d) @ %.2f MHz",
			cpuInfo[0].ModelName,
			runtime.NumCPU(),
			cpuInfo[0].Mhz)
		info.Performance.CPUFrequency = cpuInfo[0].Mhz
	}

	// Get CPU usage
	cpuPercent, _ := cpu.Percent(0, true)
	if len(cpuPercent) > 0 {
		info.Performance.CoreLoads = cpuPercent
		var total float64
		for _, p := range cpuPercent {
			total += p
		}
		info.Performance.CPUUsage = total / float64(len(cpuPercent))
	}

	// Get memory info
	memInfo, _ := mem.VirtualMemory()
	if memInfo != nil {
		info.System.Memory = fmt.Sprintf("%s / %s (%.1f%%)",
			formatBytes(memInfo.Used),
			formatBytes(memInfo.Total),
			memInfo.UsedPercent)
		info.Performance.MemoryUsage = memInfo.UsedPercent
	}

	// Get disk info
	diskInfo, _ := disk.Usage("/")
	if diskInfo != nil {
		info.System.DiskUsage = fmt.Sprintf("%s / %s (%.1f%%)",
			formatBytes(diskInfo.Used),
			formatBytes(diskInfo.Total),
			diskInfo.UsedPercent)
	}

	// Get local IP
	info.System.LocalIP = executeCommand("hostname", "-I")

	// Set colors for frontend styling
	info.Colors = Colors{
		Primary:   "\x1b[38;2;0;255;0m",   // Bright green
		Secondary: "\x1b[38;2;0;200;0m",   // Medium green
		Accent:    "\x1b[38;2;50;255;50m", // Light green
	}

	// Set ASCII art
	info.ASCIIArt = getASCIIArt(hostInfo.Platform)

	// Get load average
	loadAvg, _ := cpu.Percent(time.Second, false)
	if len(loadAvg) > 0 {
		info.System.LoadAverage = loadAvg
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"data":   info,
	})
}
