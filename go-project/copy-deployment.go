package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	defaultNamespace := "namespace-test"
	defaultDeploymentName := "deployment-test"

	// Parsing degli argomenti da linea di comando
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

	// Verifica la connessione al cluster
	if err := runCommandSilent("kubectl", "cluster-info"); err != nil {
		fmt.Println("Errore: impossibile connettersi al cluster Kubernetes.")
		os.Exit(1)
	}

	// Verifica se il namespace esiste nel cluster
	if err := runCommandSilent("kubectl", "get", "namespace", namespace); err != nil {
		fmt.Printf("Errore: il namespace '%s' non esiste.", namespace)
		os.Exit(1)
	}

	// Verifica se il deployment originale esiste
	if err := runCommandSilent("kubectl", "get", "deployment", deploymentName, "-n", namespace); err != nil {
		fmt.Printf("Errore: il deployment '%s' non esiste nel namespace '%s'.", deploymentName, namespace)
		os.Exit(1)
	}

	// Verifica se il nuovo deployment esiste già
	exists, err := deploymentExists(newDeploymentName, namespace)
	if err != nil {
		fmt.Printf("Errore durante la verifica dell'esistenza del deployment: %v", err)
		os.Exit(1)
	}
	if exists {
		reader := bufio.NewReader(os.Stdin)
		fmt.Printf("Il deployment '%s' esiste già. Inserisci un nuovo suffisso: ", newDeploymentName)
		newSuffix, _ := reader.ReadString('\n')
		newSuffix = strings.TrimSpace(newSuffix)
		newDeploymentName = deploymentName + "-test-debug-" + newSuffix

		// Ricontrolla se il nuovo nome esiste già
		exists, err := deploymentExists(newDeploymentName, namespace)
		if err != nil {
			fmt.Printf("Errore durante la verifica dell'esistenza del deployment: %v", err)
			os.Exit(1)
		}
		if exists {
			fmt.Printf("Errore: impossibile procedere. Il deployment con nome '%s' esiste già.", newDeploymentName)
			os.Exit(1)
		}
	}

	// Estrai il deployment originale in formato YAML
	if err := runCommandAndWriteToFile("kubectl", []string{"get", "deployment", deploymentName, "-n", namespace, "-o", "yaml"}, "original-deployment.yaml"); err != nil {
		fmt.Printf("Errore durante l'estrazione del deployment originale: %v", err)
		os.Exit(1)
	}

	// Rimuovi campi gestiti dal sistema utilizzando yq
	if err := runCommand("yq", "e", "del(.metadata.uid, .metadata.resourceVersion, .metadata.creationTimestamp, .metadata.generation, .metadata.managedFields)", "-i", "original-deployment.yaml"); err != nil {
		fmt.Println("Errore durante la modifica del deployment con yq.")
		os.Exit(1)
	}

	// Modifica il nome del deployment e il numero di repliche
	if err := runCommand("yq", "e", fmt.Sprintf(".spec.replicas = 1"), "-i", "original-deployment.yaml"); err != nil {
		fmt.Println("Errore durante la modifica del numero di repliche.")
		os.Exit(1)
	}
	if err := runCommand("yq", "e", fmt.Sprintf(".metadata.name = \"%s\"", newDeploymentName), "-i", "original-deployment.yaml"); err != nil {
		fmt.Println("Errore durante la modifica del nome del deployment.")
		os.Exit(1)
	}

	// Applica il nuovo deployment
	if err := runCommandSilent("kubectl", "apply", "-f", "original-deployment.yaml", "-n", namespace); err != nil {
		fmt.Println("Errore: si è verificato un problema durante l'applicazione del nuovo deployment.")
		os.Remove("original-deployment.yaml")
		os.Exit(1)
	}

	// Pulizia del file temporaneo
	os.Remove("original-deployment.yaml")
	fmt.Printf("Deployment duplicato con successo come '%s' nel namespace '%s'.", newDeploymentName, namespace)
}

func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	// cmd.Stdout = os.Stdout
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
