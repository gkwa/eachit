package run

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
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
	err = json.Unmarshal(output, &containers)
	if err != nil {
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
	fmt.Printf("Removing container: %s\n", name)
	cmd := exec.Command("incus", "rm", "--force", name)
	if err := cmd.Run(); err != nil {
		fmt.Printf("Error removing container %s: %v\n", name, err)
	}
}

func BuildHclFiles(excludeHcls, hclFiles []string) {
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

	failedBuilds := make([]string, 0)

	for _, hclFile := range buildFiles {
		if contains(excludeHcls, hclFile) {
			continue
		}

		logFile := fmt.Sprintf("%s.log", hclFile)
		logFileWriter, err := os.Create(logFile)
		if err != nil {
			fmt.Printf("Error creating log file %s: %v\n", logFile, err)
			continue
		}
		defer logFileWriter.Close()

		fmt.Printf("Building HCL file: %s\n", hclFile)

		startTime := time.Now()
		cmd := exec.Command("packer", "build", hclFile)
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

		err = cmd.Start()
		if err != nil {
			fmt.Printf("Error starting command: %v\n", err)
			continue
		}

		go io.Copy(os.Stdout, stdout)
		go io.Copy(os.Stderr, stderr)
		go io.Copy(logFileWriter, io.MultiReader(stdout, stderr))

		err = cmd.Wait()
		duration := time.Since(startTime)

		if err != nil {
			fmt.Printf("Error building HCL file %s: %v\n", hclFile, err)
			failedBuilds = append(failedBuilds, hclFile)
			log.Printf("Build duration for %s: %s (failed)\n", hclFile, duration.String())
			_, _ = logFileWriter.WriteString(fmt.Sprintf("Build duration: %s (failed)\n", duration.String()))
		} else {
			log.Printf("Build duration for %s: %s (success)\n", hclFile, duration.String())
			_, _ = logFileWriter.WriteString(fmt.Sprintf("Build duration: %s (success)\n", duration.String()))
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
