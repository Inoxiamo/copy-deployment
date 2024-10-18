package deployment

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Execute runs the deployment duplication process.
//
// The function takes two optional command-line arguments:
//
// -n <namespace>: the namespace where the deployment resides. Defaults to "namespace-test".
//
// -d <deployment_name>: the name of the deployment to duplicate. Defaults to "deployment-test".
//
// The function will check that the namespace and deployment exist in the cluster.
// If the deployment already exists, the function will prompt the user to enter a new suffix.
// The function will then extract the original deployment in YAML format, remove system-managed fields,
// modify the deployment name and replica count, apply the new deployment, and clean up the temporary file.
func Execute() {
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
			if i < len(args) {
				namespace = args[i]
			} else {
				fmt.Println("Error: missing value for -n")
				os.Exit(1)
			}
		case "-d":
			i++
			if i < len(args) {
				deploymentName = args[i]
			} else {
				fmt.Println("Error: missing value for -d")
				os.Exit(1)
			}
		}
	}

	newDeploymentName := deploymentName + "-test-debug"

	// Check if the namespace exists in the cluster
	if err := RunCommandSilent("kubectl", "get", "namespace", namespace); err != nil {
		fmt.Printf("Error: the namespace '%s' does not exist.\n", namespace)
		os.Exit(1)
	}

	// Check if the original deployment exists
	if err := RunCommandSilent("kubectl", "get", "deployment", deploymentName, "-n", namespace); err != nil {
		fmt.Printf("Error: the deployment '%s' does not exist in namespace '%s'.\n", deploymentName, namespace)
		os.Exit(1)
	}

	// Check if the new deployment already exists in the cluster
	exists, err := DeploymentExists(newDeploymentName, namespace)
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
		exists, err := DeploymentExists(newDeploymentName, namespace)
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
	if !CheckYqInstalled() {
		fmt.Println("yq not found. Starting installation...")
		err := InstallYq()
		if err != nil {
			fmt.Printf("Error during yq installation: %v\n", err)
			os.Exit(1)
		}
	}

	// Extract the original deployment in YAML format
	if err := RunCommandAndWriteToFile("kubectl", []string{"get", "deployment", deploymentName, "-n", namespace, "-o", "yaml"}, "original-deployment.yaml"); err != nil {
		fmt.Printf("Error extracting the original deployment: %v\n", err)
		os.Exit(1)
	}

	// Remove system-managed fields using yq
	if err := RunCommand("yq", "e", "del(.metadata.uid, .metadata.resourceVersion, .metadata.creationTimestamp, .metadata.generation, .metadata.managedFields)", "-i", "original-deployment.yaml"); err != nil {
		fmt.Println("Error modifying the deployment with yq.")
		os.Exit(1)
	}

	// Modify the deployment name and replica count
	if err := RunCommand("yq", "e", fmt.Sprintf(".spec.replicas = 1"), "-i", "original-deployment.yaml"); err != nil {
		fmt.Println("Error modifying the replica count.")
		os.Exit(1)
	}
	if err := RunCommand("yq", "e", fmt.Sprintf(".metadata.name = \"%s\"", newDeploymentName), "-i", "original-deployment.yaml"); err != nil {
		fmt.Println("Error modifying the deployment name.")
		os.Exit(1)
	}

	// Apply the new deployment
	if err := RunCommandSilent("kubectl", "apply", "-f", "original-deployment.yaml", "-n", namespace); err != nil {
		fmt.Println("Error: there was a problem applying the new deployment.")
		os.Remove("original-deployment.yaml")
		os.Exit(1)
	}

	// Clean up the temporary file
	os.Remove("original-deployment.yaml")
	fmt.Printf("Deployment successfully duplicated as '%s' in namespace '%s'.\n", newDeploymentName, namespace)
}
