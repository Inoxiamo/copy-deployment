package deployment

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
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
// -t <tag_image>: the tag of the image to use for the new deployment. Defaults to an empty string.
//
// -s <secret_data>: the data of the secret to use for the new deployment. Defaults to an empty string.
//
// The function will check that the namespace and deployment exist in the cluster.
// If the deployment already exists, the function will prompt the user to enter a new suffix.
// The function will then extract the original deployment in YAML format, remove system-managed fields,
// modify the deployment name and replica count, apply the new deployment, and clean up the temporary file.
func Execute() {
	defaultNamespace := "namespace-test"
	defaultDeploymentName := "deployment-test"
	tagImage := ""
	secretData := ""

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
		case "-t":
			i++
			if i < len(args) {
				tagImage = args[i]
			}
		case "-s":
			i++
			if i < len(args) {
				secretData = args[i]
			} else {
				fmt.Println("Errore: valore mancante per -s")
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
	if err := RunCommand("yq", "e", "del(.metadata.uid, .metadata.resourceVersion, .metadata.creationTimestamp, .metadata.generation, .metadata.managedFields, .spec.template.spec.containers[].livenessProbe, .spec.template.spec.containers[].readinessProbe)", "-i", "original-deployment.yaml"); err != nil {
		fmt.Println("Error modifying the deployment with yq.")
		os.Exit(1)
	}

	// Modify the deployment name and replica count
	if err := RunCommand("yq", "e", fmt.Sprintf(".spec.replicas = 1"), "-i", "original-deployment.yaml"); err != nil {
		fmt.Println("Error modifying the replica count.")
		os.Exit(1)
	}
	if err := RunCommand("yq", "e",
		fmt.Sprintf(
			`.metadata.name = "%s" | .spec.selector.matchLabels.app = "%s" | .spec.template.metadata.labels.app = "%s"`,
			newDeploymentName, newDeploymentName, newDeploymentName),
		"-i", "original-deployment.yaml"); err != nil {
		fmt.Println("Error modifying the deployment name and labels.")
		os.Exit(1)
	}

	// Assume tagImage is already defined and contains the tag value
	if tagImage != "" {
		// Retrieve the current image name
		output, err := exec.Command("yq", "e", ".spec.template.spec.containers[0].image", "original-deployment.yaml").CombinedOutput()
		if err != nil {
			fmt.Printf("Error retrieving the image name: %s\n", err)
			os.Exit(1)
		}
		currentImage := strings.TrimSpace(string(output))

		// Split the image name to separate into parts
		parts := strings.Split(currentImage, ":")
		if len(parts) < 2 {
			fmt.Println("The image name format is unexpected. It should contain two ':' characters separating the registry, name, and tag.")
			os.Exit(1)
		}

		// Rebuild the image name with the new tag, keeping registry and name the same
		parts[len(parts)-1] = tagImage // Replace only the last part, the tag
		newImage := strings.Join(parts, ":")

		// Update the image in the YAML file
		if err := exec.Command("yq", "e", fmt.Sprintf(".spec.template.spec.containers[0].image = \"%s\"", newImage), "-i", "original-deployment.yaml").Run(); err != nil {
			fmt.Println("Error modifying the image tag:", err)
			os.Exit(1)
		}

		fmt.Println("Image tag updated successfully.")
	}

	if secretData != "" {
		// Parse the key-value pairs of the secrets
		secretDataMap := parseSecretData(secretData)

		// Retrieve the names of the secrets used by the deployment
		secretNames, err := getSecretNamesFromDeployment("original-deployment.yaml")
		if err != nil {
			fmt.Printf("Error retrieving the names of the secrets used by the deployment: %v\n", err)
			os.Exit(1)
		}

		// For each secret, create a new secret with the modified data
		for _, secretName := range secretNames {
			newSecretName := secretName + "-" + newDeploymentName

			// Retrieve the YAML of the original secret
			err = RunCommandAndWriteToFile("kubectl", []string{"get", "secret", secretName, "-n", namespace, "-o", "yaml"}, "original-secret.yaml")
			if err != nil {
				fmt.Printf("Error retrieving the secret '%s': %v\n", secretName, err)
				os.Exit(1)
			}

			// Remove system-managed fields from the secret YAML
			err = RunCommand("yq", "e", "del(.metadata.uid, .metadata.resourceVersion, .metadata.creationTimestamp, .metadata.annotations, .metadata.ownerReferences)", "-i", "original-secret.yaml")
			if err != nil {
				fmt.Println("Error cleaning the secret YAML.")
				os.Exit(1)
			}

			// Modify the secret data
			err = modifySecretData("original-secret.yaml", secretDataMap)
			if err != nil {
				fmt.Printf("Error modifying the secret '%s': %v\n", secretName, err)
				os.Exit(1)
			}

			// Change the secret name to the new name
			err = RunCommand("yq", "e", fmt.Sprintf(".metadata.name = \"%s\"", newSecretName), "-i", "original-secret.yaml")
			if err != nil {
				fmt.Printf("Error updating the secret name: %v\n", err)
				os.Exit(1)
			}

			// Apply the new secret
			err = RunCommandSilent("kubectl", "apply", "-f", "original-secret.yaml", "-n", namespace)
			if err != nil {
				fmt.Printf("Error applying the new secret '%s': %v\n", newSecretName, err)
				os.Exit(1)
			}

			// Update the deployment YAML to use the new secret
			err = updateDeploymentToUseNewSecret("original-deployment.yaml", secretName, newSecretName)
			if err != nil {
				fmt.Printf("Error updating the deployment to use the new secret: %v\n", err)
				os.Exit(1)
			}

			// Remove the temporary secret YAML file
			os.Remove("original-secret.yaml")
		}
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
