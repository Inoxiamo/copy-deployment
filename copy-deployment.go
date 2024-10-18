package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func main() {
	defaultNamespace := "namespace-test"
	defaultDeploymentName := "deployment-test"

	// Parsing command-line arguments
	namespace := defaultNamespace
	deploymentName := defaultDeploymentName

	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-n":
			i++
			namespace = args[i]
		case "-d":
			i++
			deploymentName = args[i]
		}
	}

	newDeploymentName := deploymentName + "-test-debug"

	// Check if the namespace exists in the cluster
	if err := runCommandSilent("kubectl", "get", "namespace", namespace); err != nil {
		fmt.Printf("Error: the namespace '%s' does not exist.\n", namespace)
		os.Exit(1)
	}

	// Check if the original deployment exists
	if err := runCommandSilent("kubectl", "get", "deployment", deploymentName, "-n", namespace); err != nil {
		fmt.Printf("Error: the deployment '%s' does not exist in namespace '%s'.\n", deploymentName, namespace)
		os.Exit(1)
	}

	// Check if the new deployment already exists in the cluster
	exists, err := deploymentExists(newDeploymentName, namespace)
	if err != nil {
		fmt.Printf("Error while checking deployment existence: %v\n", err)
		os.Exit(1)
	}
	if exists {
		reader := bufio.NewReader(os.Stdin)
		fmt.Printf("The deployment '%s' already exists. Please enter a new suffix: ", newDeploymentName)
		newSuffix, _ := reader.ReadString('\n')
		newSuffix = strings.TrimSpace(newSuffix)
		newDeploymentName = deploymentName + "-test-debug-" + newSuffix

		// Re-check if the new name already exists
		exists, err := deploymentExists(newDeploymentName, namespace)
		if err != nil {
			fmt.Printf("Error while checking deployment existence: %v\n", err)
			os.Exit(1)
		}
		if exists {
			fmt.Printf("Error: cannot proceed. The deployment named '%s' already exists.\n", newDeploymentName)
			os.Exit(1)
		}
	}

	// Check if yq is installed
	if !checkYqInstalled() {
		fmt.Println("yq not found. Starting installation...")
		err := installYq()
		if err != nil {
			fmt.Printf("Error during yq installation: %v\n", err)
			os.Exit(1)
		}
	}

	// Extract the original deployment in YAML format
	if err := runCommandAndWriteToFile("kubectl", []string{"get", "deployment", deploymentName, "-n", namespace, "-o", "yaml"}, "original-deployment.yaml"); err != nil {
		fmt.Printf("Error extracting the original deployment: %v\n", err)
		os.Exit(1)
	}

	// Remove system-managed fields using yq
	if err := runCommand("yq", "e", "del(.metadata.uid, .metadata.resourceVersion, .metadata.creationTimestamp, .metadata.generation, .metadata.managedFields)", "-i", "original-deployment.yaml"); err != nil {
		fmt.Println("Error modifying the deployment with yq.")
		os.Exit(1)
	}

	// Modify the deployment name and replica count
	if err := runCommand("yq", "e", fmt.Sprintf(".spec.replicas = 1"), "-i", "original-deployment.yaml"); err != nil {
		fmt.Println("Error modifying the replica count.")
		os.Exit(1)
	}
	if err := runCommand("yq", "e", fmt.Sprintf(".metadata.name = \"%s\"", newDeploymentName), "-i", "original-deployment.yaml"); err != nil {
		fmt.Println("Error modifying the deployment name.")
		os.Exit(1)
	}

	// Apply the new deployment
	if err := runCommandSilent("kubectl", "apply", "-f", "original-deployment.yaml", "-n", namespace); err != nil {
		fmt.Println("Error: there was a problem applying the new deployment.")
		os.Remove("original-deployment.yaml")
		os.Exit(1)
	}

	// Clean up the temporary file
	os.Remove("original-deployment.yaml")
	fmt.Printf("Deployment successfully duplicated as '%s' in namespace '%s'.\n", newDeploymentName, namespace)
}

func checkYqInstalled() bool {
	_, err := exec.LookPath("yq")
	return err == nil
}

func installYq() error {
	osName := runtime.GOOS
	arch := runtime.GOARCH
	var downloadURL string

	// Determine the appropriate download URL
	switch osName {
	case "linux":
		downloadURL = fmt.Sprintf("https://github.com/mikefarah/yq/releases/latest/download/yq_%s_%s", osName, arch)
	case "darwin":
		downloadURL = fmt.Sprintf("https://github.com/mikefarah/yq/releases/latest/download/yq_%s_%s", osName, arch)
	case "windows":
		downloadURL = fmt.Sprintf("https://github.com/mikefarah/yq/releases/latest/download/yq_%s_%s.exe", osName, arch)
	default:
		return fmt.Errorf("Unsupported operating system: %s", osName)
	}

	// Download the file
	fmt.Println("Downloading yq...")
	resp, err := http.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("Error downloading yq: %v", err)
	}
	defer resp.Body.Close()

	// Check that the download was successful
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Download failed with status code: %d", resp.StatusCode)
	}

	// Determine the installation path
	var binPath string
	if osName == "windows" {
		binPath = filepath.Join(os.TempDir(), "yq.exe")
	} else {
		binPath = filepath.Join(os.TempDir(), "yq")
	}

	// Create the file
	outFile, err := os.Create(binPath)
	if err != nil {
		return fmt.Errorf("Error creating yq file: %v", err)
	}
	defer outFile.Close()

	// Copy the content to the file
	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		return fmt.Errorf("Error saving yq: %v", err)
	}

	// Make the file executable
	if osName != "windows" {
		err = os.Chmod(binPath, 0755)
		if err != nil {
			return fmt.Errorf("Error changing yq permissions: %v", err)
		}
	}

	// Add the temporary directory to PATH for this execution
	os.Setenv("PATH", fmt.Sprintf("%s%c%s", filepath.Dir(binPath), os.PathListSeparator, os.Getenv("PATH")))

	fmt.Println("yq successfully installed.")

	return nil
}

func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runCommandSilent(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stderr = nil
	return cmd.Run()
}

func deploymentExists(deploymentName, namespace string) (bool, error) {
	cmd := exec.Command("kubectl", "get", "deployment", deploymentName, "-n", namespace)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return false, err
	}

	if err := cmd.Start(); err != nil {
		return false, err
	}

	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		text := scanner.Text()
		if strings.Contains(text, "Error from server (NotFound)") {
			return false, nil
		}
		fmt.Fprintln(os.Stderr, text)
	}

	if err := cmd.Wait(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() == 1 {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func runCommandAndWriteToFile(name string, args []string, filename string) error {
	cmd := exec.Command(name, args...)
	output, err := cmd.Output()
	if err != nil {
		return err
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(output)
	if err != nil {
		return err
	}

	return nil
}
