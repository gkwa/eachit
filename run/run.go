package run

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/taylormonacelli/eachit/ntfy"
)

type Container struct {
	Name string `json:"name"`
}

func DestroyContainers(containerNamesToRemove []string) {
	cmd := exec.Command("incus", "ls", "--format=json")
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("Error listing containers:", err)
		return
	}

	var containers []Container
	if err := json.Unmarshal(output, &containers); err != nil {
		fmt.Println("Error parsing JSON:", err)
		return
	}

	for _, container := range containers {
		if contains(containerNamesToRemove, container.Name) {
			destroyContainer(container.Name)
		}
	}
}

func destroyContainer(name string) {
	maxRetries := 10
	retryDelay := time.Second

	for i := 0; i < maxRetries; i++ {
		cmd := exec.Command("incus", "ls", name)
		if err := cmd.Run(); err != nil {
			if exitError, ok := err.(*exec.ExitError); ok {
				if exitError.ExitCode() == 1 {
					fmt.Printf("Container %s does not exist\n", name)
					return
				}
			}
			fmt.Printf("Error checking container %s: %v\n", name, err)
			if i < maxRetries-1 {
				fmt.Printf("Retrying in %s...\n", retryDelay)
				time.Sleep(retryDelay)
				continue
			}
		}

		fmt.Printf("Removing container: %s\n", name)
		cmd = exec.Command("incus", "rm", "--force", name)
		if err := cmd.Run(); err != nil {
			fmt.Printf("Error removing container %s: %v\n", name, err)
			if i < maxRetries-1 {
				fmt.Printf("Retrying in %s...\n", retryDelay)
				time.Sleep(retryDelay)
				continue
			}
		}
		return
	}
	fmt.Printf("Failed to remove container %s after %d retries\n", name, maxRetries)
}

func BuildHclFiles(containerNamesToRemove, excludeHcls, hclFiles []string) {
	var buildFiles []string

	if len(hclFiles) > 0 {
		buildFiles = hclFiles
	} else {
		var err error
		buildFiles, err = filepath.Glob("*.hcl")
		if err != nil {
			fmt.Println("Error finding HCL files:", err)
			return
		}
	}

	successFile := "success_list.txt"
	failureFile := "failure_list.txt"

	failedBuilds := make([]string, 0)

	tempDir := os.TempDir()

	for _, hclFile := range buildFiles {
		if contains(excludeHcls, hclFile) {
			continue
		}

		logFile := fmt.Sprintf("%s.log", hclFile)
		if fileInfo, err := os.Stat(logFile); err == nil {
			if fileInfo.Size() > 5 {
				fmt.Printf("Log file %s already exists, skipping build\n", logFile)
				continue
			}
		}

		DestroyContainers(containerNamesToRemove)

		fmt.Printf("Running packer init for HCL file: %s\n", hclFile)
		initCmd := exec.Command("packer", "init", hclFile)
		if err := initCmd.Run(); err != nil {
			fmt.Printf("Error running packer init for HCL file %s: %v\n", hclFile, err)
			os.Exit(1)
		}

		tempLogFile := filepath.Join(tempDir, filepath.Base(logFile))
		tempLogFileWriter, err := os.Create(tempLogFile)
		if err != nil {
			fmt.Printf("Error creating temp log file %s: %v\n", tempLogFile, err)
			continue
		}
		defer tempLogFileWriter.Close()

		fmt.Printf("Building HCL file: %s\n", hclFile)

		startTime := time.Now()
		cmd := exec.Command("packer", "build", "-color=false", hclFile)
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			fmt.Printf("Error getting stdout pipe: %v\n", err)
			continue
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			fmt.Printf("Error getting stderr pipe: %v\n", err)
			continue
		}

		if err := cmd.Start(); err != nil {
			fmt.Printf("Error starting command: %v\n", err)
			continue
		}

		go func() {
			_, err := io.Copy(io.MultiWriter(os.Stdout, tempLogFileWriter), stdout)
			if err != nil {
				fmt.Printf("Error copying stdout: %v\n", err)
			}
		}()
		go func() {
			_, err := io.Copy(io.MultiWriter(os.Stderr, tempLogFileWriter), stderr)
			if err != nil {
				fmt.Printf("Error copying stderr: %v\n", err)
			}
		}()

		err = cmd.Wait()
		buildDuration := time.Since(startTime).Truncate(time.Second).String()

		if err != nil {
			fmt.Printf("Error building HCL file %s: %v\n", hclFile, err)
			failedBuilds = append(failedBuilds, hclFile)
			log.Printf("Build duration for %s: %s (failed)\n", hclFile, buildDuration)
			_, _ = tempLogFileWriter.WriteString(fmt.Sprintf("Build duration: %s (failed)\n", buildDuration))

			// Remove from success list if it exists there, then add to failure list
			if removeFromFile(successFile, hclFile) {
				slog.Debug("moved build from success to failure list", "file", hclFile)
			}
			addToFile(failureFile, hclFile)
		} else {
			log.Printf("Build duration for %s: %s (success)\n", hclFile, buildDuration)
			_, _ = tempLogFileWriter.WriteString(fmt.Sprintf("Build duration: %s (success)\n", buildDuration))

			// Remove from failure list if it exists there, then add to success list
			if removeFromFile(failureFile, hclFile) {
				slog.Debug("moved build from failure to success list", "file", hclFile)
			}
			addToFile(successFile, hclFile)
		}

		if err := os.Rename(tempLogFile, logFile); err != nil {
			fmt.Printf("Error moving log file from %s to %s: %v\n", tempLogFile, logFile, err)
		}

		logFileBytes, err := os.ReadFile(logFile)
		if err != nil {
			fmt.Printf("Error reading log file %s: %v\n", logFile, err)
		} else {
			ntfy.SendNotification(string(logFileBytes))
		}
	}

	if len(failedBuilds) > 0 {
		fmt.Println("Failed builds:")
		for _, build := range failedBuilds {
			fmt.Println(build)
		}
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if strings.TrimSpace(s) == strings.TrimSpace(item) {
			return true
		}
	}
	return false
}

// addToFile adds content to file if it doesn't already exist
func addToFile(filename, content string) {
	if fileContains(filename, content) {
		slog.Debug("entry already exists in file", "file", filename, "content", content)
		return
	}

	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		fmt.Printf("Error appending to file %s: %v\n", filename, err)
		return
	}
	defer file.Close()

	_, err = file.WriteString(content + "\n")
	if err != nil {
		fmt.Printf("Error writing to file %s: %v\n", filename, err)
		return
	}

	slog.Debug("added entry to file", "file", filename, "content", content)
}

func fileContains(filename, content string) bool {
	file, err := os.Open(filename)
	if err != nil {
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) == strings.TrimSpace(content) {
			return true
		}
	}
	return false
}

// removeFromFile removes content from file and returns true if content was found and removed
func removeFromFile(filename, content string) bool {
	if !fileContains(filename, content) {
		slog.Debug("entry not found in file, nothing to remove", "file", filename, "content", content)
		return false
	}

	file, err := os.Open(filename)
	if err != nil {
		slog.Debug("could not open file for reading", "file", filename, "error", err)
		return false
	}

	var lines []string
	scanner := bufio.NewScanner(file)
	found := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == strings.TrimSpace(content) {
			found = true
			continue // skip this line (remove it)
		}
		lines = append(lines, line)
	}
	file.Close()

	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading file %s: %v\n", filename, err)
		return false
	}

	if !found {
		slog.Debug("content not found in file during removal", "file", filename, "content", content)
		return false
	}

	// Write the updated content back to the file
	file, err = os.OpenFile(filename, os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		fmt.Printf("Error opening file for writing %s: %v\n", filename, err)
		return false
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, line := range lines {
		if _, err := writer.WriteString(line + "\n"); err != nil {
			fmt.Printf("Error writing to file: %v\n", err)
			return false
		}
	}

	if err := writer.Flush(); err != nil {
		fmt.Printf("Error flushing file: %v\n", err)
		return false
	}

	slog.Debug("removed entry from file", "file", filename, "content", content)
	return true
}

// Deprecated: use addToFile instead
func appendToFile(filename, content string) {
	addToFile(filename, content)
}
