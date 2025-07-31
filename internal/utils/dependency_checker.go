package utils

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// DependencyChecker checks required external dependencies
type DependencyChecker struct {
	dependencies    map[string][]string // map dep -> version check command args
	DependencyPaths map[string]string   // Exported: Changed to uppercase 'D'
}

// NewDependencyChecker creates a new dependency checker with commands to check dependencies
func NewDependencyChecker() *DependencyChecker {
	return &DependencyChecker{
		dependencies: map[string][]string{
			"yt-dlp":  {"yt-dlp", "--version"},
			"aria2c":  {"aria2c", "--version"},
			"ffmpeg":  {"ffmpeg", "-version"},
			"ffprobe": {"ffprobe", "-version"}, // Added ffprobe as a dependency to check
		},
		DependencyPaths: make(map[string]string), // Initialize the new map, use exported name
	}
}

// CheckDependencies checks if dependencies are installed by verifying binary exists and command runs
// It now also populates the DependencyPaths map.
func (dc *DependencyChecker) CheckDependencies() (map[string]bool, error) {
	results := make(map[string]bool)
	missing := []string{}

	// Workaround Termux PATH to include ~/.local/bin where pip installs binaries
	if runtime.GOOS == "android" && isTermux() {
		localBin := fmt.Sprintf("%s/.local/bin", os.Getenv("HOME"))
		path := os.Getenv("PATH")
		if !strings.Contains(path, localBin) {
			os.Setenv("PATH", path+":"+localBin)
		}
	}

	for dep, args := range dc.dependencies {
		ok, path, err := dc.checkDependency(args) // Modified to return path
		results[dep] = ok
		if ok {
			dc.DependencyPaths[dep] = path // Store the found path, use exported name
		}
		if err != nil || !ok {
			missing = append(missing, dep)
		}
	}

	if len(missing) > 0 {
		return results, fmt.Errorf("missing dependencies: %s", strings.Join(missing, ", "))
	}

	return results, nil
}

// GetDependencyPaths returns the map of found dependency executable paths.
// This is a public getter method to access the exported field.
func (dc *DependencyChecker) GetDependencyPaths() map[string]string {
	return dc.DependencyPaths
}

// checkDependency runs the version command to verify dependency presence with timeout context
// It now returns the absolute path of the found binary.
func (dc *DependencyChecker) checkDependency(args []string) (bool, string, error) { // Modified return signature
	if len(args) == 0 {
		return false, "", errors.New("no command specified")
	}

	binary := args[0]
	var foundPath string

	// Try locating the binary via 'which' (or 'where' on Windows)
	var whichCmd *exec.Cmd
	if runtime.GOOS == "windows" {
		whichCmd = exec.Command("where", binary)
	} else {
		whichCmd = exec.Command("which", binary)
	}

	out, err := whichCmd.Output()
	if err == nil {
		path := strings.TrimSpace(string(out))
		if path != "" {
			// On Windows, 'where' can return multiple paths. Take the first one.
			if runtime.GOOS == "windows" {
				paths := strings.Split(path, "\n")
				if len(paths) > 0 {
					path = strings.TrimSpace(paths[0])
				}
			}
			if _, err := os.Stat(path); err == nil {
				foundPath = path
			}
		}
	}

	// Fallback or confirm: use exec.LookPath on PATH
	if foundPath == "" {
		path, err := exec.LookPath(binary)
		if err != nil {
			return false, "", fmt.Errorf("binary %s not found in PATH", binary)
		}
		foundPath = path
	}

	// Run version command with timeout context using the foundPath
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, foundPath, args[1:]...) // Use foundPath here
	output, err := cmd.CombinedOutput()
	if err != nil {
		// On Windows, if the command runs but the output indicates an error (e.g., specific exit codes),
		// we might need more nuanced checks. For now, rely on `CombinedOutput` error.
		return false, "", fmt.Errorf("command failed: %w, output: %s", err, string(output))
	}

	return true, foundPath, nil // Return the found path
}

// InstallDependencies installs missing dependencies based on OS/distro
func (dc *DependencyChecker) InstallDependencies() error {
	results, _ := dc.CheckDependencies() // Re-check to get latest missing deps
	missing := []string{}
	for dep, ok := range results {
		if !ok {
			missing = append(missing, dep)
		}
	}

	if len(missing) == 0 {
		fmt.Println("All dependencies already installed.")
		return nil
	}

	osType := runtime.GOOS
	fmt.Printf("Detected OS: %s\n", osType)

	switch osType {
	case "android":
		if isTermux() {
			fmt.Println("Detected Android with Termux environment")
			if err := dc.installOnPkg(missing); err != nil {
				return err
			}
			if contains(missing, "yt-dlp") {
				return dc.installYtDlpWithPip()
			}
			return nil
		}
		return fmt.Errorf("unsupported OS: android (outside Termux not supported)")

	case "linux":
		if isTermux() { // Termux on Linux is still Android under the hood
			fmt.Println("Detected Termux environment on Linux")
			if err := dc.installOnPkg(missing); err != nil {
				return err
			}
			if contains(missing, "yt-dlp") {
				return dc.installYtDlpWithPip()
			}
			return nil
		}

		distro, err := detectLinuxDistro()
		if err != nil {
			return err
		}
		fmt.Printf("Detected Linux distro: %s\n", distro)

		switch distro {
		case "debian", "ubuntu","kali":
			if err := dc.installOnApt(missing); err != nil {
				return err
			}
		case "centos", "fedora", "rhel":
			if err := dc.installOnYum(missing); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported Linux distro: %s", distro)
		}

		if contains(missing, "yt-dlp") {
			return dc.installYtDlpWithPip()
		}

	case "darwin":
		if err := dc.installOnBrew(missing); err != nil {
			return err
		}

	case "windows": // <<< ADDED WINDOWS SUPPORT
		fmt.Println("Detected Windows OS.")
		if err := dc.installOnChocolatey(missing); err != nil {
			return err
		}
		// yt-dlp on Windows is also installed via pip
		if contains(missing, "yt-dlp") {
			return dc.installYtDlpWithPip()
		}
		return nil

	default:
		return fmt.Errorf("unsupported OS: %s", osType)
	}

	return nil
}

// installOnApt installs packages via apt for Debian/Ubuntu
func (dc *DependencyChecker) installOnApt(deps []string) error {
	fmt.Println("Updating apt package lists...")
	if err := runCommand("apt", []string{"update", "-y"}); err != nil {
		return fmt.Errorf("apt update failed: %w", err)
	}

	for _, dep := range deps {
		if dep == "yt-dlp" {
			continue // yt-dlp is installed via pip
		}
		pkgName := mapDepToPkg(dep)
		if pkgName == "" { // Handle cases where mapDepToPkg might return empty for non-system deps
			continue
		}
		fmt.Printf("Installing %s via apt...\n", pkgName)
		if err := runCommand("apt ", []string{"install", "-y", pkgName}); err != nil {
			return fmt.Errorf("failed to install %s: %w", pkgName, err)
		}
	}

	return nil
}

// installOnYum installs packages via yum for RedHat/CentOS/Fedora
func (dc *DependencyChecker) installOnYum(deps []string) error {
	fmt.Println("Updating yum package lists...")
	if err := runCommand("yum", []string{"makecache"}); err != nil {
		return fmt.Errorf("yum makecache failed: %w", err)
	}

	for _, dep := range deps {
		if dep == "yt-dlp" {
			continue // yt-dlp is installed via pip
		}
		pkgName := mapDepToPkg(dep)
		if pkgName == "" {
			continue
		}
		fmt.Printf("Installing %s via yum...\n", pkgName)
		if err := runCommand("yum", []string{"install", "-y", pkgName}); err != nil {
			return fmt.Errorf("failed to install %s: %w", pkgName, err)
		}
	}

	return nil
}

// installOnBrew installs packages via brew for macOS
func (dc *DependencyChecker) installOnBrew(deps []string) error {
	for _, dep := range deps {
		pkgName := mapDepToPkg(dep)
		if pkgName == "" {
			continue
		}
		fmt.Printf("Installing %s via brew...\n", pkgName)
		if err := runCommand("brew", []string{"install", pkgName}); err != nil {
			return fmt.Errorf("failed to install %s: %w", pkgName, err)
		}
	}
	return nil
}

// installOnPkg installs packages using Termux pkg manager
func (dc *DependencyChecker) installOnPkg(deps []string) error {
	fmt.Println("Updating package lists...")
	if err := runCommand("pkg", []string{"update", "-y"}); err != nil {
		return fmt.Errorf("pkg update failed: %w", err)
	}

	for _, dep := range deps {
		if dep == "yt-dlp" {
			continue // yt-dlp is installed via pip
		}
		pkgName := mapDepToPkgTermux(dep)
		if pkgName == "" {
			continue
		}
		fmt.Printf("Installing %s via pkg...\n", pkgName)
		if err := runCommand("pkg", []string{"install", "-y", pkgName}); err != nil {
			return fmt.Errorf("failed to install %s: %w", pkgName, err)
		}
	}

	return nil
}

// installOnChocolatey installs packages via Chocolatey for Windows
func (dc *DependencyChecker) installOnChocolatey(deps []string) error {
	// First, check if Chocolatey itself is installed.
	// We'll use a specific check for choco that doesn't rely on the main checkDependency for simplicity here,
	// as it's a prerequisite.
	_, chocoPath, err := dc.checkDependency([]string{"choco", "--version"})
	if err != nil {
		fmt.Println("Chocolatey (choco) not found. Please install Chocolatey first from https://chocolatey.org/install.")
		return errors.New("Chocolatey is not installed or not in PATH")
	} else {
		fmt.Printf("Chocolatey found at: %s\n", chocoPath)
	}

	for _, dep := range deps {
		if dep == "yt-dlp" {
			continue // yt-dlp is installed via pip even on Windows
		}
		pkgName := mapDepToPkgWindows(dep)
		if pkgName == "" {
			continue
		}
		fmt.Printf("Installing %s via Chocolatey...\n", pkgName)
		// Note: Many Chocolatey installations require administrative privileges.
		// Inform the user about this.
		fmt.Println("Note: This step might require administrative privileges. If it fails, please run your application as administrator.")
		if err := runCommand("choco", []string{"install", "-y", pkgName}); err != nil {
			return fmt.Errorf("failed to install %s via Chocolatey: %w", pkgName, err)
		}
	}
	return nil
}

// installYtDlpWithPip installs yt-dlp using python3 -m pip
func (dc *DependencyChecker) installYtDlpWithPip() error {
	fmt.Println("Installing yt-dlp with pip...")

	// Use python3 or python based on system availability
	pythonCmd := "python3"
	if _, err := exec.LookPath("python3"); err != nil {
		pythonCmd = "python" // Fallback to 'python' if 'python3' isn't found
		if _, err := exec.LookPath("python"); err != nil {
			return errors.New("Neither 'python3' nor 'python' found. Python is required to install yt-dlp.")
		}
	}

	cmd := exec.Command(pythonCmd, "-m", "pip", "install", "--upgrade", "yt-dlp")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("command pip failed: %w", err)
	}
	return nil
}

// mapDepToPkg maps dependencies to OS package names (for apt, yum, brew)
func mapDepToPkg(dep string) string {
	switch dep {
	case "aria2c":
		return "aria2"
	case "ffmpeg":
		return "ffmpeg"
	case "ffprobe": // Add ffprobe mapping
		return "ffmpeg" // ffprobe usually comes with ffmpeg
	default:
		return dep
	}
}

// mapDepToPkgTermux maps dependencies to Termux package names
func mapDepToPkgTermux(dep string) string {
	switch dep {
	case "aria2c":
		return "aria2"
	case "ffmpeg":
		return "ffmpeg"
	case "ffprobe": // Add ffprobe mapping for Termux
		return "ffmpeg" // ffprobe usually comes with ffmpeg
	default:
		return dep
	}
}

// mapDepToPkgWindows maps dependencies to Windows package names (e.g., Chocolatey)
func mapDepToPkgWindows(dep string) string {
	switch dep {
	case "aria2c":
		return "aria2" // Common Chocolatey package name
	case "ffmpeg":
		return "ffmpeg" // Common Chocolatey package name
	case "ffprobe":
		return "ffmpeg" // ffprobe usually comes with ffmpeg
	default:
		return dep
	}
}

// detectLinuxDistro reads /etc/os-release to detect Linux distro ID
func detectLinuxDistro() (string, error) {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return "", fmt.Errorf("failed to read /etc/os-release: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "ID=") {
			id := strings.TrimPrefix(line, "ID=")
			id = strings.Trim(id, `"`)
			return strings.ToLower(id), nil
		}
	}

	return "", errors.New("linux distro ID not found in /etc/os-release")
}

// isTermux checks if the environment is Termux by env var PREFIX
func isTermux() bool {
	prefix := os.Getenv("PREFIX")
	return strings.Contains(prefix, "com.termux")
}

// runCommand executes a system command with 1 minute timeout and streams output
func runCommand(command string, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("Running command: %s %s\n", command, strings.Join(args, " "))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("command %s failed: %w", command, err)
	}

	return nil
}

// contains utility for string slice
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
