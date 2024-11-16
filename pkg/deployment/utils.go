package deployment

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
)

// RunCommand runs a command with the given arguments and returns an error if the
// command fails.  The command's stderr is connected to os.Stderr.
func RunCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// RunCommandSilent runs a command with the given arguments and returns an error
// if the command fails.  The command's stderr is discarded.
func RunCommandSilent(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stderr = nil
	return cmd.Run()
}

// DeploymentExists checks if a Kubernetes deployment with the given name exists
// in the specified namespace. It runs the "kubectl get deployment" command and
// parses the output to determine existence. If the deployment is not found, it
// returns false with no error. If any other error occurs, it returns false with
// the corresponding error. If the deployment exists, it returns true.
func DeploymentExists(deploymentName, namespace string) (bool, error) {
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

// RunCommandAndWriteToFile runs the given command and writes its output to the specified file.
// If a file with the given name already exists, it will be overwritten.
// If the command fails, the function will return the command's error.
func RunCommandAndWriteToFile(name string, args []string, filename string) error {
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
func parseSecretData(secretData string) map[string]string {
	data := make(map[string]string)
	pairs := strings.Split(secretData, ";")
	for _, pair := range pairs {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 {
			key := kv[0]
			value := kv[1]
			data[key] = value
		}
	}
	return data
}

func getSecretNamesFromDeployment(deploymentYaml string) ([]string, error) {
	var secretNames []string

	// Get secret names from environment variables
	cmd := exec.Command("yq", "e", ".spec.template.spec.containers[].env[].valueFrom.configMapKeyRef.name", deploymentYaml)
	output, err := cmd.Output()
	if err == nil {
		names := strings.Split(strings.TrimSpace(string(output)), "\n")
		secretNames = append(secretNames, names...)
	}

	// Get secret names from volumes
	cmd = exec.Command("yq", "e", ".spec.template.spec.volumes[].secret.secretName", deploymentYaml)
	output, err = cmd.Output()
	if err == nil {
		names := strings.Split(strings.TrimSpace(string(output)), "\n")
		secretNames = append(secretNames, names...)
	}

	// Remove duplicates
	secretMap := make(map[string]bool)
	uniqueSecretNames := []string{}
	for _, name := range secretNames {
		name = strings.TrimSpace(name)
		if name != "" && !secretMap[name] {
			secretMap[name] = true
			uniqueSecretNames = append(uniqueSecretNames, name)
		}
	}

	if len(uniqueSecretNames) == 0 {
		return nil, fmt.Errorf("No secrets found in the deployment")
	}

	return uniqueSecretNames, nil
}

// Modify the secret data in the secret YAML file.
// The function takes the path to the secret YAML file and a map of new key-value pairs.
// The function will update the secret data by merging the existing data with the new data.
// If the new data contains a key that is already present in the existing data, the old value will be overwritten.
// The function will return an error if there is an issue with reading or modifying the secret YAML file.
func modifySecretData(secretYaml string, newData map[string]string) error {
	// Extract the value of the blob base64 from the .data.config field
	cmd := exec.Command("yq", "e", ".data[\"secrets.env\"]", secretYaml)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("Error reading the blob value from the secret: %v", err)
	}

	// Remove any quotes
	base64Blob := strings.TrimSpace(strings.Trim(string(output), "\"'"))

	// Decode the base64 blob
	decodedBytes, err := base64.StdEncoding.DecodeString(base64Blob)
	if err != nil {
		return fmt.Errorf("Error decoding the base64 blob: %v", err)
	}

	// The decoded content is a set of key-value pairs
	// Determine the format of the content: for example, let's assume it's a properties file (key=value per line)
	content := string(decodedBytes)

	// Parse the content to extract the key-value pairs
	existingData := parseKeyValueContent(content)

	// Merge the new data, overwriting duplicate keys
	for key, value := range newData {
		existingData[key] = value
	}

	// Rebuild the updated content
	updatedContent := buildKeyValueContent(existingData)

	// Re-encode the updated content in base64
	updatedBase64Blob := base64.StdEncoding.EncodeToString([]byte(updatedContent))

	// Update the value of .data.config in the secret YAML file
	err = RunCommand("yq", "e", fmt.Sprintf(".data[\"secrets.env\"] = \"%s\"", updatedBase64Blob), "-i", secretYaml)
	if err != nil {
		return fmt.Errorf("Error updating the blob value in the secret: %v", err)
	}

	return nil
}

// Parse the key-value content from a string.
// The function takes a string as input and returns a map of key-value pairs.
// The function assumes that the content is a set of key-value pairs separated by equals signs and newlines.
func parseKeyValueContent(content string) map[string]string {
	data := make(map[string]string)
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Ignore empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		kv := strings.SplitN(line, "=", 2)
		if len(kv) == 2 {
			key := strings.TrimSpace(kv[0])
			value := strings.TrimSpace(kv[1])
			data[key] = value
		}
	}
	return data
}

// Build a key-value content from a map of key-value pairs.
// The function takes a map of key-value pairs as input and returns a string.
// The function assumes that the map should be formatted as a properties file (key=value per line).
func buildKeyValueContent(data map[string]string) string {
	var lines []string
	for key, value := range data {
		line := fmt.Sprintf("%s=%s", key, value)
		lines = append(lines, line)
	}
	// Sort the lines to have a consistent output (optional)
	sort.Strings(lines)
	return strings.Join(lines, "\n")
}

// Update the deployment YAML file to use a new secret.
// The function takes the path to the deployment YAML file, the name of the old secret, and the name of the new secret.
// The function will update the names of the secret in the environment variables and volumes.
// The function will return an error if there is an issue with modifying the deployment YAML file.
func updateDeploymentToUseNewSecret(deploymentYaml, oldSecretName, newSecretName string) error {
	// Update the names of the secret in the environment variables
	err := RunCommand("yq", "e", fmt.Sprintf(`(.spec.template.spec.containers[].env[].valueFrom.configMapKeyRef | select(.name == "%s")).name = "%s"`, oldSecretName, newSecretName), "-i", deploymentYaml)
	if err != nil {
		return err
	}
	// Update the names of the secret in the volumes
	err = RunCommand("yq", "e", fmt.Sprintf(`(.spec.template.spec.volumes[].secret | select(.secretName == "%s")).secretName = "%s"`, oldSecretName, newSecretName), "-i", deploymentYaml)
	if err != nil {
		return err
	}
	return nil
}
