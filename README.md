# Kubernetes Deployment Copy Script

This repository contains a Go script designed to duplicate an existing Kubernetes deployment, appending the suffix `-test-debug` to the duplicated deployment. The script creates a copy of an existing deployment while maintaining important configurations such as `serviceAccount`, `configMap`, `secrets`, and `image`, and forces the number of replicas to `1`.

## Features

- **Duplicate an Existing Deployment**: Easily create a copy of any existing Kubernetes deployment.
- **Preserve Configurations**: Maintains critical configurations like `serviceAccount`, `configMap`, `secrets`, and `image`.
- **Set Replicas to One**: Forces the replica count to `1` for the duplicated deployment to prevent resource overconsumption.
- **Adjust Deployment Name**: Appends `-test-debug` to the original deployment name to differentiate it.
- **Cross-Platform Support**: Available as a compiled executable for Linux, Windows, and macOS.

## Prerequisites

- **Kubernetes CLI (kubectl)**: Installed and configured to interact with your Kubernetes cluster.
- **Access to a Kubernetes Cluster**: Ensure you have the necessary permissions to create deployments.
- **yq Command-Line YAML Processor**: The script automatically checks for `yq` and installs it if not found.
  - *Note*: No manual installation of `yq` is required; the script handles it.

## Installation

### Option 1: Download Pre-Built Executables

Pre-built executables are available for Linux, Windows, and macOS.

- **Linux**: Download `copy-deployment-linux-amd64` from the Releases page.
- **Windows**: Download `copy-deployment-windows-amd64.exe` from the Releases page.
- **macOS**: Download `copy-deployment-darwin-amd64` from the Releases page.

Make sure to give execution permissions to the downloaded file (if necessary):

```bash
chmod +x copy-deployment
```

### Option 2: Compile from Source

#### Prerequisites

- Go installed (version 1.16 or higher).

#### Compilation Steps

1. **Clone the Repository**

   ```bash
   git clone https://github.com/your-repo.git
   cd your-repo/go-project
   ```

2. **Compile for Your Platform**

   - **Linux**:
     ```bash
     GOOS=linux GOARCH=amd64 go build -o copy-deployment
     ```
   - **Windows**:
     ```bash
     GOOS=windows GOARCH=amd64 go build -o copy-deployment.exe
     ```
   - **macOS**:
     ```bash
     GOOS=darwin GOARCH=amd64 go build -o copy-deployment
     ```

   The compiled executable will be created in the current directory.

3. **Verify the Executable**

   Ensure the executable has been created:

   ```bash
   ls -l copy-deployment*
   ```

## Usage

Run the executable with the following options:

```bash
./copy-deployment -n <namespace> -d <deployment_name> -t <tag_image>
```

- `<namespace>`: (Optional) The namespace of the deployment to duplicate. Defaults to `namespace-test`.
- `<deployment_name>`: (Optional) The name of the deployment to duplicate. Defaults to `deployment-test`.
- `<tag_image>`: (Optional) The tag of the image of the deployment. Defaults is an empty string.

### Example

To duplicate a deployment named `api` in the default namespace:

```bash
./copy-deployment -n default -d api
```

This will create a new deployment named `api-test-debug` in the default namespace, with `1` replica.

## Command-Line Arguments

- `-n`: Specify the namespace of the deployment.
- `-d`: Specify the name of the deployment.
- `-t`: [Optional] Specify the tag of the image of the deployment.

## How It Works

1. **Cluster Connection Verification**: The script checks if it can connect to your Kubernetes cluster using `kubectl`.
2. **Namespace and Deployment Validation**: Verifies the existence of the specified namespace and deployment.
3. **yq Installation Check**: Ensures `yq` is installed. If not, the script downloads and installs it temporarily.
4. **YAML Extraction**: Extracts the deployment's configuration into a YAML file, removing system-managed fields like `uid`, `resourceVersion`, and `creationTimestamp` to avoid conflicts.
5. **Modify Deployment**:
   - **Deployment Name**: Appends `-test-debug` to the original deployment name.
   - **Replica Count**: Sets the number of replicas to `1`.
6. **Apply New Deployment**: Applies the modified deployment to the cluster, creating the new deployment.
7. **Cleanup**: Removes the temporary YAML file used during the process.

## Troubleshooting

- **Permission Errors**: Ensure you have the correct permissions and that your `kubectl` context is set correctly.
- **Deployment Already Exists**: If the new deployment name already exists, the script will prompt you to enter a new suffix.
- **yq Installation Issues**: The script attempts to install `yq` if it's not found. Ensure you have internet access and the necessary permissions for the script to download and execute files.

## Requirements

- **Kubernetes Cluster**: Tested with Kubernetes v1.20+.
- **Go**: Required if you are compiling the script from source.
- **Operating System**: Compatible with Linux, Windows, and macOS.

## Notes

- **Testing Purpose**: This script is intended for testing configurations and should not be used in production without proper adjustments.
- **Deployment Only**: The script duplicates only the deployment, not associated services or other resources.

## License

This project is licensed under the MIT License.

## Contributions

Contributions are welcome! Feel free to fork this repository, open issues, or submit pull requests if you have suggestions or improvements.

## Author

Developed by Inoxiamo.

