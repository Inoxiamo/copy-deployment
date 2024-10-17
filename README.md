# Kubernetes Deployment Copy Script

This repository contains a Bash script for duplicating an existing Kubernetes deployment, appending the suffix "*-test-debug*" to the duplicated deployment. The script is designed to create a copy of an existing deployment while maintaining important configurations such as serviceAccount, configMap, secrets, and image, and forcing the number of replicas to 1.

## Features

Duplicate an existing Kubernetes deployment.

Preserve important configurations like serviceAccount, configMap, secrets, and image.

Force the replicas count to 1 for the duplicated deployment.

Adjust labels to differentiate the new deployment from the original.

## Prerequisites

To use this script, you need the following tools installed and configured on your machine:

Bash Shell: Ensure you have a Bash environment available.

Make the Script Executable: Before running the script, make it executable with:
```bash
chmod +x copy-deployment.sh
```

* Kubernetes CLI (kubectl): To interact with the Kubernetes cluster.

* yq: A lightweight and portable command-line YAML processor (version 4.x or higher).

You also need access to a Kubernetes cluster and sufficient permissions to create deployments.

Usage

```bash
./copy-deployment.sh [-n namespace] [-d deployment_name] [-l label_key]
```

Parameters
```
-n namespace (optional): The namespace of the deployment to duplicate. Defaults to da.

-d deployment_name (optional): The name of the deployment to duplicate. Defaults to api.

-l label_key (optional): The label key used for differentiating the deployment. Defaults to app.
```

### Example

To duplicate a deployment named api in the default namespace:

```bash
./copy-deployment.sh -n default -d api -l app
```

This will create a new deployment named *api-test-debug* in the default namespace, with 1 replica and modified labels to differentiate it from the original.

## How It Works

Cluster Connection Verification: The script first checks if you can connect to the Kubernetes cluster.

Namespace and Deployment Validation: It verifies the existence of the specified namespace and deployment.

YAML Extraction: The script extracts the deployment's configuration into a YAML file, removing system-managed fields such as uid, resourceVersion, and creationTimestamp to avoid conflicts.

Modify Deployment:

The new deployment name is set by appending *-test-debug* to the original name.

The number of replicas is forced to 1.

Labels are modified to distinguish the duplicated deployment.

Apply New Deployment: The modified deployment is applied to the cluster, creating a new deployment.

Cleanup: The temporary YAML file is removed.

## Troubleshooting

Permission Errors: If you encounter permission errors like write /dev/stdout: permission denied, ensure you have the correct permissions and that your kubectl context is set correctly.

Missing Labels: The script requires a label key (default: app). If the original deployment does not have this label, you will need to specify an existing label key using the -l flag.

## Requirements

Kubernetes Cluster (tested with v1.20+)

Bash Shell

yq version 4.x or higher

Notes

This script is intended for testing purposes. The duplicated deployment is meant for testing configurations and should not be used in production without proper adjustments.

The script does not duplicate Kubernetes services or other resources, only the deployment.

## License

This project is licensed under the MIT License.

Contributions

Feel free to fork this repository, open issues, or submit pull requests if you have any suggestions or improvements.

## Author

Developed by Inoxiamo