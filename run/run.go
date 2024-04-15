package run

import (
    "encoding/json"
    "fmt"
    "os/exec"
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

func contains(slice []string, item string) bool {
    for _, s := range slice {
        if s == item {
            return true
        }
    }
    return false
}

