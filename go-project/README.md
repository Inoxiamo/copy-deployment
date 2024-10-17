# Kubernetes Deployment Copy Script - Compilation and Distribution Guide

This guide explains how to compile the `copy-deployment` Go script and create distributable executables for Linux, Windows, and macOS platforms. The compiled executables will be placed in their respective distribution folders.

## Prerequisites

- [Go](https://golang.org/doc/install) installed on your system.
- Basic knowledge of using terminal commands.

## Compilation Steps

1. **Create Distribution Folders**

   First, create the distribution folders for Linux, Windows, and macOS using the following command:

   ```bash
   mkdir -p dist/linux dist/windows dist/mac
   ```

2. **Compile for Each Platform**

   Use the `go build` command to create executables for each platform:

   - **Linux**:

     ```bash
     GOOS=linux GOARCH=amd64 go build -o dist/linux/copy-deployment
     ```

   - **Windows**:

     ```bash
     GOOS=windows GOARCH=amd64 go build -o dist/windows/copy-deployment.exe
     ```

   - **macOS**:

     ```bash
     GOOS=darwin GOARCH=amd64 go build -o dist/mac/copy-deployment
     ```

   These commands will generate an executable for each target operating system and place them in the appropriate distribution folders.

3. **Verify the Executables**

   After compiling, verify that the executables have been created successfully:

   ```bash
   ls dist/linux/
   ls dist/windows/
   ls dist/mac/
   ```

   You should see the respective `copy-deployment` executable in each folder.

## Distribution

You can now distribute the compiled executables:

- **Linux**: Share the executable located in `dist/linux/`
- **Windows**: Share the executable located in `dist/windows/`
- **macOS**: Share the executable located in `dist/mac/`





Each executable can be run directly on the corresponding platform without requiring additional compilation.

## Running the Executable

To run the executable, use the following command:

```bash
./copy-deployment -n <namespace> -d <deployment_name>
```

Replace `<namespace>` and `<deployment_name>` with the appropriate values for your Kubernetes environment.
