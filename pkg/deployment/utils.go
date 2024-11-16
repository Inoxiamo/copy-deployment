package deployment

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
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
		kv := strings.SplitN(pair, ":", 2)
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

	// Ottiene i nomi dei secret dalle variabili d'ambiente
	cmd := exec.Command("yq", "e", ".spec.template.spec.containers[].env[].valueFrom.secretKeyRef.name", deploymentYaml)
	output, err := cmd.Output()
	if err == nil {
		names := strings.Split(strings.TrimSpace(string(output)), "\n")
		secretNames = append(secretNames, names...)
	}

	// Ottiene i nomi dei secret dai volumi
	cmd = exec.Command("yq", "e", ".spec.template.spec.volumes[].secret.secretName", deploymentYaml)
	output, err = cmd.Output()
	if err == nil {
		names := strings.Split(strings.TrimSpace(string(output)), "\n")
		secretNames = append(secretNames, names...)
	}

	// Rimuove duplicati
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
		return nil, fmt.Errorf("Nessun secret trovato nel deployment")
	}

	return uniqueSecretNames, nil
}

func modifySecretData(secretYaml string, newData map[string]string) error {
	for key, value := range newData {
		// Codifica in base64 il valore
		encodedValue := base64.StdEncoding.EncodeToString([]byte(value))
		// Aggiorna o aggiunge il campo data
		err := RunCommand("yq", "e", fmt.Sprintf(".data.%s = \"%s\"", key, encodedValue), "-i", secretYaml)
		if err != nil {
			return err
		}
	}
	return nil
}

func updateDeploymentToUseNewSecret(deploymentYaml, oldSecretName, newSecretName string) error {
	// Aggiorna i nomi dei secret nelle variabili d'ambiente
	err := RunCommand("yq", "e", fmt.Sprintf(`(.spec.template.spec.containers[].env[].valueFrom.secretKeyRef | select(.name == "%s")).name = "%s"`, oldSecretName, newSecretName), "-i", deploymentYaml)
	if err != nil {
		return err
	}
	// Aggiorna i nomi dei secret nei volumi
	err = RunCommand("yq", "e", fmt.Sprintf(`(.spec.template.spec.volumes[].secret | select(.secretName == "%s")).secretName = "%s"`, oldSecretName, newSecretName), "-i", deploymentYaml)
	if err != nil {
		return err
	}
	return nil
}
