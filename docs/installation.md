# DataScrapexter Installation Guide

## System Requirements

DataScrapexter requires a modern operating system with Go runtime support. The application has been tested on Linux (Ubuntu 20.04+, CentOS 8+, Debian 10+), macOS (10.15 Catalina and later), and Windows (Windows 10 version 1909 and later). Your system should have at least 2 GB of RAM for basic operations, though 4 GB or more is recommended for large-scale scraping operations. Storage requirements are minimal for the application itself, requiring only 50 MB, but you should allocate sufficient space for scraped data storage based on your expected usage.

The primary prerequisite is Go version 1.24 or later. DataScrapexter leverages modern Go features for performance and reliability, making this version requirement essential. Additionally, Git is required for source installation, and Make is recommended for simplified building and installation processes.

## Installation Methods

DataScrapexter offers multiple installation approaches to accommodate different user preferences and system configurations. Each method provides access to the full functionality of the application, allowing you to choose the approach that best fits your workflow and environment.

### Installing from Source

Installing from source provides the most flexibility and ensures you have the latest features and fixes. This method is recommended for developers and users who want to contribute to the project or customize the build process.

Begin by cloning the repository from GitHub:

```bash
git clone https://github.com/valpere/DataScrapexter.git
cd DataScrapexter
```

Once you have the source code, install the required dependencies using Go modules:

```bash
go mod download
```

The project includes a comprehensive Makefile that simplifies the build process. To build the application with optimizations:

```bash
make build
```

This command creates an optimized binary in the `./bin` directory. The build process embeds version information and strips unnecessary symbols to reduce file size while maintaining performance.

For system-wide installation, use the install target:

```bash
sudo make install
```

This command copies the binary to `/usr/local/bin` by default, making it accessible from any directory. If you prefer a different installation location, you can modify the `GOPATH` environment variable or manually copy the binary to your preferred location.

### Using Go Install

For users familiar with Go tooling, the `go install` command provides a streamlined installation process:

```bash
go install github.com/valpere/DataScrapexter/cmd/datascrapexter@latest
```

This method downloads the source code, compiles it with your local Go installation, and places the binary in your `$GOPATH/bin` directory. Ensure this directory is in your system's PATH for easy access to the `datascrapexter` command.

### Binary Releases

Pre-compiled binaries offer the fastest installation path for users who prefer not to build from source. These binaries are available for major platforms and architectures, thoroughly tested before release to ensure stability and compatibility.

To install using binary releases, navigate to the GitHub releases page at https://github.com/valpere/DataScrapexter/releases. Select the latest release version and download the appropriate archive for your operating system and architecture. The naming convention follows the pattern `datascrapexter-version-platform-architecture`.

For Linux systems on AMD64 architecture:

```bash
wget https://github.com/valpere/DataScrapexter/releases/download/v0.1.0/datascrapexter-v0.1.0-linux-amd64.tar.gz
tar -xzf datascrapexter-v0.1.0-linux-amd64.tar.gz
sudo mv datascrapexter /usr/local/bin/
sudo chmod +x /usr/local/bin/datascrapexter
```

For macOS systems with Apple Silicon:

```bash
curl -L https://github.com/valpere/DataScrapexter/releases/download/v0.1.0/datascrapexter-v0.1.0-darwin-arm64.tar.gz -o datascrapexter.tar.gz
tar -xzf datascrapexter.tar.gz
sudo mv datascrapexter /usr/local/bin/
sudo chmod +x /usr/local/bin/datascrapexter
```

Windows users should download the `.zip` archive and extract the executable to a directory in their PATH, such as `C:\Program Files\DataScrapexter`. Remember to add this directory to your system's PATH environment variable for command-line access.

### Docker Installation

Docker provides consistent execution environments across different systems, making it ideal for production deployments and team environments. DataScrapexter's Docker images include all necessary dependencies and are optimized for minimal size while maintaining full functionality.

Pull the latest Docker image from the GitHub Container Registry:

```bash
docker pull ghcr.io/valpere/datascrapexter:latest
```

To run DataScrapexter using Docker with local configuration files:

```bash
docker run --rm -it \
  -v $(pwd)/configs:/app/configs \
  -v $(pwd)/outputs:/app/outputs \
  ghcr.io/valpere/datascrapexter:latest run /app/configs/example.yaml
```

For frequent Docker usage, consider creating an alias in your shell configuration:

```bash
alias datascrapexter='docker run --rm -it -v $(pwd):/app/workspace ghcr.io/valpere/datascrapexter:latest'
```

This alias allows you to use DataScrapexter commands as if it were installed locally while maintaining the isolation benefits of containerization.

### Package Managers

While DataScrapexter is not yet available in major package repositories, we are working on submissions to popular package managers. Future releases will support installation through Homebrew for macOS users, APT and YUM repositories for Linux distributions, and Chocolatey for Windows systems. Check the project's GitHub page for updates on package manager availability.

## Post-Installation Setup

After successful installation, verify that DataScrapexter is properly configured by checking the version:

```bash
datascrapexter version
```

This command displays version information, build time, and git commit hash, confirming successful installation and providing important debugging information for support requests.

### Configuration Directory Setup

DataScrapexter benefits from organized directory structures for configurations and outputs. Create the recommended directory structure:

```bash
mkdir -p ~/datascrapexter/{configs,outputs,logs,scripts}
```

This structure separates different aspects of your scraping projects, making management easier as your usage grows. The configs directory stores YAML configuration files, outputs contains scraped data, logs maintains operational records, and scripts holds automation utilities.

### Environment Configuration

Certain DataScrapexter features benefit from environment variable configuration. Create a `.env` file in your project directory for sensitive information:

```bash
# DataScrapexter Environment Configuration
export DATASCRAPEXTER_HOME="$HOME/datascrapexter"
export DATASCRAPEXTER_LOG_LEVEL="info"
export DATASCRAPEXTER_CONFIG_PATH="$DATASCRAPEXTER_HOME/configs"
export DATASCRAPEXTER_OUTPUT_PATH="$DATASCRAPEXTER_HOME/outputs"

# Proxy configuration (if needed)
# export HTTP_PROXY="http://proxy.example.com:8080"
# export HTTPS_PROXY="http://proxy.example.com:8080"

# API keys for services
# export CAPTCHA_API_KEY="your-2captcha-key"
# export PROXY_API_KEY="your-proxy-service-key"
```

Source this file in your shell configuration to make these variables available:

```bash
echo "source ~/datascrapexter/.env" >> ~/.bashrc  # For Bash
echo "source ~/datascrapexter/.env" >> ~/.zshrc   # For Zsh
```

### Initial Configuration Test

Validate your installation with a simple test scrape:

```bash
# Generate a basic configuration
datascrapexter template > test-config.yaml

# Run the test scrape
datascrapexter run test-config.yaml

# Check the output
cat output.json
```

Successful execution confirms that DataScrapexter is properly installed and can perform basic scraping operations.

## Platform-Specific Considerations

### Linux Installation Notes

Linux systems may require additional dependencies for optimal performance. Install development tools and libraries that enhance DataScrapexter's capabilities:

```bash
# Debian/Ubuntu
sudo apt-get update
sudo apt-get install build-essential ca-certificates curl

# RHEL/CentOS/Fedora
sudo yum groupinstall "Development Tools"
sudo yum install ca-certificates curl
```

For headless server environments, ensure proper locale settings to handle international content correctly:

```bash
export LANG=en_US.UTF-8
export LC_ALL=en_US.UTF-8
```

### macOS Installation Notes

macOS users should ensure Xcode Command Line Tools are installed for compilation support:

```bash
xcode-select --install
```

If you encounter certificate verification issues, update your certificate bundle:

```bash
brew install ca-certificates
```

For Apple Silicon Macs, DataScrapexter runs natively without Rosetta 2 emulation, providing optimal performance. Ensure you download the `darwin-arm64` binary or build from source with native Go installation.

### Windows Installation Notes

Windows users should consider using Windows Terminal or PowerShell for better command-line experience. The traditional Command Prompt works but may have limitations with colored output and special characters.

Add DataScrapexter to your PATH through System Properties:

1. Open System Properties (Win + Pause/Break)
2. Click "Advanced system settings"
3. Click "Environment Variables"
4. Under "System variables", find and select "Path", then click "Edit"
5. Add the directory containing datascrapexter.exe
6. Click "OK" to save changes
7. Restart your terminal for changes to take effect

For long-running scraping operations on Windows, consider using Task Scheduler instead of keeping terminal windows open.

## Troubleshooting Installation Issues

### Common Installation Problems

If you encounter "command not found" errors after installation, verify that the installation directory is in your PATH. Use `which datascrapexter` on Unix-like systems or `where datascrapexter` on Windows to locate the binary.

Permission denied errors during installation typically indicate insufficient privileges. Use `sudo` for system-wide installation or install to a user-writable directory and update your PATH accordingly.

### Building from Source Issues

Compilation errors often result from incompatible Go versions. Verify your Go installation:

```bash
go version
```

If the version is below 1.24, update Go following the official installation guide at https://golang.org/doc/install.

Missing dependencies manifest as import errors during compilation. Ensure you run `go mod download` before building, and that your internet connection allows access to Go module repositories.

### Verifying Installation Integrity

For binary downloads, verify checksums to ensure download integrity:

```bash
# Download checksums file
wget https://github.com/valpere/DataScrapexter/releases/download/v0.1.0/checksums.txt

# Verify checksum (Linux/macOS)
sha256sum -c checksums.txt 2>&1 | grep OK

# Verify checksum (Windows PowerShell)
Get-FileHash datascrapexter.exe -Algorithm SHA256
```

## Updating DataScrapexter

Regular updates ensure access to new features, performance improvements, and security fixes. The update process depends on your installation method.

For source installations, pull the latest changes and rebuild:

```bash
cd /path/to/DataScrapexter
git pull origin main
make clean
make build
sudo make install
```

For Go installations, use the install command with the latest tag:

```bash
go install github.com/valpere/DataScrapexter/cmd/datascrapexter@latest
```

Docker users should pull the latest image:

```bash
docker pull ghcr.io/valpere/datascrapexter:latest
```

Always review release notes before updating to understand new features and potential breaking changes. Backup your configurations and test updates in non-production environments first.

## Uninstalling DataScrapexter

Should you need to remove DataScrapexter, the process varies by installation method.

For manual installations, remove the binary:

```bash
sudo rm /usr/local/bin/datascrapexter
```

For Go installations, remove from GOPATH:

```bash
rm $GOPATH/bin/datascrapexter
```

Clean up configuration and data directories if no longer needed:

```bash
rm -rf ~/datascrapexter  # Careful: This removes all configurations and data
```

Docker users can remove images:

```bash
docker rmi ghcr.io/valpere/datascrapexter:latest
```

## Getting Help

If you encounter issues during installation, several resources are available. The GitHub Issues page provides a searchable database of known issues and solutions. The community Discord server offers real-time assistance from experienced users. For commercial support requirements, contact support@datascrapexter.com.

When reporting installation issues, include your operating system and version, Go version (if applicable), exact error messages, and steps taken to reproduce the problem. This information helps maintainers and community members provide accurate assistance.

## Next Steps

With DataScrapexter successfully installed, explore the User Guide to understand core concepts and workflows. The Configuration Reference provides detailed information about all available options. Example configurations in the `examples/` directory demonstrate common use cases and best practices.

Begin with simple scraping tasks to familiarize yourself with DataScrapexter's operation before progressing to complex configurations. The modular design allows you to start basic and add sophistication as your needs evolve.

Remember to always respect website terms of service and implement appropriate rate limiting. Responsible scraping ensures sustainable access to data while maintaining positive relationships with content providers.
