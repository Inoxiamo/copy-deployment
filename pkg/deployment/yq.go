package deployment

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// CheckYqInstalled checks if yq is installed on the system.
//
// It returns true if yq is installed and false otherwise.
func CheckYqInstalled() bool {
	_, err := exec.LookPath("yq")
	return err == nil
}

// InstallYq installs yq on the system if it is not already installed.
//
// The function will download the yq binary from the GitHub releases page and
// install it in the temporary directory. It will then add the temporary directory
// to the PATH environment variable so that the yq command can be executed.
//
// The function returns nil if the installation is successful and an error if
// there are any issues during the installation process.
func InstallYq() error {
	osName := runtime.GOOS
	arch := runtime.GOARCH
	var downloadURL string

	// Determina l'URL di download appropriato
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

	// Scarica il file
	fmt.Println("Downloading yq...")
	resp, err := http.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("Error downloading yq: %v", err)
	}
	defer resp.Body.Close()

	// Verifica che il download sia andato a buon fine
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Download failed with status code: %d", resp.StatusCode)
	}

	// Determina il percorso di installazione
	var binPath string
	if osName == "windows" {
		binPath = filepath.Join(os.TempDir(), "yq.exe")
	} else {
		binPath = filepath.Join(os.TempDir(), "yq")
	}

	// Crea il file
	outFile, err := os.Create(binPath)
	if err != nil {
		return fmt.Errorf("Error creating yq file: %v", err)
	}
	defer outFile.Close()

	// Copia il contenuto nel file
	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		return fmt.Errorf("Error saving yq: %v", err)
	}

	// Rende il file eseguibile
	if osName != "windows" {
		err = os.Chmod(binPath, 0755)
		if err != nil {
			return fmt.Errorf("Error changing yq permissions: %v", err)
		}
	}

	// Aggiunge la directory temporanea al PATH per questa esecuzione
	os.Setenv("PATH", fmt.Sprintf("%s%c%s", filepath.Dir(binPath), os.PathListSeparator, os.Getenv("PATH")))

	fmt.Println("yq successfully installed.")

	return nil
}
