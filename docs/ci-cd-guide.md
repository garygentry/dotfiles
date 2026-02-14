# CI/CD and IaC Integration Guide

This guide covers using the dotfiles system in automated environments: CI/CD pipelines, Infrastructure as Code (IaC), container images, and configuration management tools.

## Overview

The `--unattended` flag enables fully automated, zero-prompt installation suitable for:

- **Infrastructure as Code**: Terraform, CloudFormation, Pulumi
- **Container Images**: Docker, Podman
- **Configuration Management**: Ansible, Chef, Puppet
- **CI/CD Pipelines**: GitHub Actions, GitLab CI, Jenkins
- **VM Provisioning**: Packer, Vagrant

### How It Works

When `--unattended` is set:

- ✅ All interactive prompts are skipped
- ✅ Default values are used for all module configurations
- ✅ Secrets authentication is automatically skipped
- ✅ Confirmation prompts are bypassed
- ✅ Auto-detection works for non-interactive environments

The system automatically enables unattended mode when stdin is not interactive (e.g., `curl | bash`).

## Quick Start

### Basic Unattended Installation

```bash
# Using bootstrap script
curl -sfL https://raw.githubusercontent.com/garygentry/dotfiles/main/bootstrap.sh | bash -s -- --unattended

# Direct installation
git clone https://github.com/garygentry/dotfiles.git ~/.dotfiles
cd ~/.dotfiles
go build -o bin/dotfiles .
./bin/dotfiles install --unattended
```

### With Specific Profile

```bash
# Create config.yml before installation
cat > ~/.dotfiles/config.yml <<EOF
profile: minimal
secrets:
  provider: ""
EOF

# Run installation
dotfiles install --unattended
```

## Use Cases

### Terraform / AWS CloudFormation

**EC2 User Data Script:**

```bash
#!/bin/bash
set -euo pipefail

# Install dependencies
apt-get update
apt-get install -y git golang-go

# Clone and install dotfiles
export USER=ubuntu
export HOME=/home/ubuntu
cd $HOME
git clone https://github.com/garygentry/dotfiles.git .dotfiles
cd .dotfiles

# Build and run
go build -o bin/dotfiles .
./bin/dotfiles install --unattended --profile server

# Verify installation
./bin/dotfiles status
```

**Terraform Example:**

```hcl
resource "aws_instance" "server" {
  ami           = "ami-0c55b159cbfafe1f0"
  instance_type = "t3.micro"

  user_data = <<-EOF
    #!/bin/bash
    curl -sfL https://raw.githubusercontent.com/garygentry/dotfiles/main/bootstrap.sh | \
      sudo -u ubuntu bash -s -- --unattended --profile server
  EOF

  tags = {
    Name = "dotfiles-provisioned-server"
  }
}
```

### Docker Images

**Dockerfile Example:**

```dockerfile
FROM ubuntu:22.04

# Install dependencies
RUN apt-get update && apt-get install -y \
    git \
    golang-go \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Create user
RUN useradd -m -s /bin/bash developer
USER developer
WORKDIR /home/developer

# Clone dotfiles
RUN git clone https://github.com/garygentry/dotfiles.git .dotfiles
WORKDIR /home/developer/.dotfiles

# Build CLI
RUN go build -o bin/dotfiles .

# Install dotfiles in unattended mode
RUN ./bin/dotfiles install --unattended --profile minimal --skip-failed

# Set PATH
ENV PATH="/home/developer/.dotfiles/bin:${PATH}"

CMD ["/bin/bash"]
```

**Multi-stage Build (Optimized):**

```dockerfile
# Build stage
FROM golang:1.22-alpine AS builder
WORKDIR /build
COPY . .
RUN go build -o dotfiles .

# Runtime stage
FROM ubuntu:22.04
RUN apt-get update && apt-get install -y git && rm -rf /var/lib/apt/lists/*
RUN useradd -m -s /bin/bash developer
USER developer
WORKDIR /home/developer

COPY --from=builder /build /home/developer/.dotfiles
WORKDIR /home/developer/.dotfiles

RUN ./dotfiles install --unattended --profile docker --skip-failed

ENV PATH="/home/developer/.dotfiles/bin:${PATH}"
CMD ["/bin/bash"]
```

### Ansible Playbooks

**Basic Playbook:**

```yaml
---
- name: Install dotfiles
  hosts: servers
  become: yes
  become_user: "{{ target_user }}"

  tasks:
    - name: Install dependencies
      apt:
        name:
          - git
          - golang-go
        state: present
      become_user: root

    - name: Clone dotfiles repository
      git:
        repo: https://github.com/garygentry/dotfiles.git
        dest: "~/.dotfiles"
        version: main

    - name: Build dotfiles CLI
      command: go build -o bin/dotfiles .
      args:
        chdir: "~/.dotfiles"
        creates: "~/.dotfiles/bin/dotfiles"

    - name: Install dotfiles
      command: ./bin/dotfiles install --unattended --profile {{ dotfiles_profile | default('default') }}
      args:
        chdir: "~/.dotfiles"
      register: dotfiles_install
      changed_when: "'succeeded' in dotfiles_install.stdout"

    - name: Verify installation
      command: ./bin/dotfiles status
      args:
        chdir: "~/.dotfiles"
      changed_when: false
```

**With Role Structure:**

```yaml
# roles/dotfiles/tasks/main.yml
---
- name: Ensure dependencies
  apt:
    name: [git, golang-go]
    state: present
  become: yes

- name: Clone dotfiles
  git:
    repo: "{{ dotfiles_repo }}"
    dest: "{{ ansible_env.HOME }}/.dotfiles"
    version: "{{ dotfiles_version | default('main') }}"

- name: Build CLI
  command: go build -o bin/dotfiles .
  args:
    chdir: "{{ ansible_env.HOME }}/.dotfiles"
    creates: "{{ ansible_env.HOME }}/.dotfiles/bin/dotfiles"

- name: Create config.yml
  template:
    src: config.yml.j2
    dest: "{{ ansible_env.HOME }}/.dotfiles/config.yml"
  when: dotfiles_config is defined

- name: Install modules
  command: >
    ./bin/dotfiles install --unattended
    {{ '--profile ' + dotfiles_profile if dotfiles_profile is defined else '' }}
    {{ '--skip-failed' if dotfiles_skip_failed | default(false) else '' }}
  args:
    chdir: "{{ ansible_env.HOME }}/.dotfiles"
```

### GitHub Actions

```yaml
name: Test Dotfiles Installation

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:
  test-install:
    runs-on: ubuntu-latest

    steps:
      - name: Checkout repository
        uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.22'

      - name: Build dotfiles
        run: go build -o bin/dotfiles .

      - name: Test unattended installation
        run: |
          ./bin/dotfiles install --unattended --dry-run
          ./bin/dotfiles install --unattended --profile minimal --skip-failed

      - name: Verify installation
        run: ./bin/dotfiles status

      - name: Test uninstall
        run: ./bin/dotfiles uninstall git --unattended --dry-run
```

### Packer Templates

**HCL2 Format:**

```hcl
packer {
  required_plugins {
    amazon = {
      version = ">= 1.0.0"
      source  = "github.com/hashicorp/amazon"
    }
  }
}

source "amazon-ebs" "dotfiles" {
  ami_name      = "dotfiles-${formatdate("YYYY-MM-DD-hhmm", timestamp())}"
  instance_type = "t3.micro"
  region        = "us-east-1"
  source_ami_filter {
    filters = {
      name                = "ubuntu/images/*ubuntu-jammy-22.04-amd64-server-*"
      root-device-type    = "ebs"
      virtualization-type = "hvm"
    }
    most_recent = true
    owners      = ["099720109477"]
  }
  ssh_username = "ubuntu"
}

build {
  sources = ["source.amazon-ebs.dotfiles"]

  provisioner "shell" {
    inline = [
      "sudo apt-get update",
      "sudo apt-get install -y git golang-go",
      "git clone https://github.com/garygentry/dotfiles.git ~/.dotfiles",
      "cd ~/.dotfiles && go build -o bin/dotfiles .",
      "./bin/dotfiles install --unattended --profile server",
      "./bin/dotfiles status"
    ]
  }
}
```

## Configuration

### Pre-creating config.yml

Create configuration before installation to control profile and settings:

```bash
# Create config file
cat > ~/.dotfiles/config.yml <<EOF
profile: minimal
secrets:
  provider: ""  # Disable secrets in CI/CD
EOF

# Run installation
dotfiles install --unattended
```

### Using Profiles

Create custom profiles for different environments:

```bash
# profiles/ci.yml
- git
- zsh

# profiles/server.yml
- git
- tmux
- zsh

# profiles/docker.yml
- git
- neovim
```

Use with `--profile` flag:

```bash
dotfiles install --unattended --profile ci
```

### Environment Variables

The system uses environment variables for configuration:

```bash
# Set dotfiles directory (default: ~/.dotfiles)
export DOTFILES_DIR=/opt/dotfiles

# Run installation
dotfiles install --unattended
```

## Handling Failures

### Skip Failed Modules

Continue installation even if some modules fail:

```bash
dotfiles install --unattended --skip-failed
```

This is essential for container images where some modules (e.g., GUI tools) may not be compatible.

### Fail Fast

Stop immediately on first failure (useful for testing):

```bash
dotfiles install --unattended --fail-fast
```

### Dry Run

Preview what would happen without making changes:

```bash
dotfiles install --unattended --dry-run
```

### Error Handling Example

```bash
#!/bin/bash
set -euo pipefail

# Function to handle errors
install_dotfiles() {
  if ! dotfiles install --unattended --skip-failed; then
    echo "ERROR: Dotfiles installation failed"
    dotfiles status  # Show what succeeded
    exit 1
  fi
}

# Run with error handling
install_dotfiles

# Verify critical modules
if ! dotfiles status | grep -q "git.*installed"; then
  echo "ERROR: Critical module 'git' not installed"
  exit 1
fi

echo "Dotfiles installed successfully"
```

## Secrets Management

### Skipping Secrets

In unattended mode, secrets authentication is automatically skipped:

```bash
# Secrets authentication is skipped automatically
dotfiles install --unattended
```

### Pre-authenticating 1Password

For environments where secrets are needed:

```bash
# Authenticate before running dotfiles
op account add --address my.1password.com --email user@example.com
op signin

# Run installation
dotfiles install --unattended
```

### Using Profiles Without Secrets

Create a profile that excludes secrets-dependent modules:

```bash
# profiles/no-secrets.yml
- git
- zsh
- tmux
# Note: 'ssh' module is excluded (requires 1password)
```

## Verification

### Exit Codes

The dotfiles CLI uses standard exit codes:

- `0`: Success
- `1`: Failure

```bash
#!/bin/bash
if dotfiles install --unattended; then
  echo "Installation successful"
else
  echo "Installation failed with exit code $?"
  exit 1
fi
```

### Status Checks

Verify installation state:

```bash
# Check overall status
dotfiles status

# Check specific module
dotfiles status | grep git

# Programmatic check
if dotfiles status | grep -q "git.*installed"; then
  echo "Git module is installed"
fi
```

### Logging

Enable verbose logging for debugging:

```bash
# Verbose output
dotfiles install --unattended --verbose

# JSON logging (for log aggregation)
dotfiles install --unattended --log-json
```

## Best Practices

### 1. Use Profiles

Create environment-specific profiles:

```bash
# Development
profiles/dev.yml

# Production servers
profiles/prod.yml

# CI/CD
profiles/ci.yml

# Docker containers
profiles/docker.yml
```

### 2. Test with Dry Run

Always test in dry-run mode first:

```bash
# Test installation
dotfiles install --unattended --profile prod --dry-run

# Review plan, then run
dotfiles install --unattended --profile prod
```

### 3. Use Skip Failed in Containers

Container environments may not support all modules:

```bash
# Dockerfile
RUN ./bin/dotfiles install --unattended --skip-failed
```

### 4. Version Pin Your Dotfiles

Use specific Git tags or commits:

```bash
# Terraform user_data
git clone https://github.com/garygentry/dotfiles.git
git checkout v1.0.0
```

### 5. Capture Logs

Save installation logs for debugging:

```bash
# Save verbose logs
dotfiles install --unattended --verbose > /var/log/dotfiles-install.log 2>&1

# JSON logs for aggregation
dotfiles install --unattended --log-json > /var/log/dotfiles.json
```

### 6. Verify After Installation

Always check status after installation:

```bash
#!/bin/bash
set -euo pipefail

dotfiles install --unattended --skip-failed
dotfiles status

# Verify critical modules
for module in git zsh; do
  if ! dotfiles status | grep -q "${module}.*installed"; then
    echo "ERROR: ${module} not installed"
    exit 1
  fi
done
```

## Troubleshooting

### Installation Hangs

**Problem**: Installation blocks waiting for input

**Solution**:
```bash
# Ensure --unattended is set
dotfiles install --unattended

# Check if stdin is being piped
echo "Installing..." | dotfiles install --unattended
```

### Modules Fail in Docker

**Problem**: Some modules require interactive terminal or system features

**Solution**:
```bash
# Use --skip-failed to continue
dotfiles install --unattended --skip-failed

# Or create a docker-specific profile
dotfiles install --unattended --profile docker
```

### Secrets Not Available

**Problem**: 1Password prompts block installation

**Solution**:
```bash
# Unattended mode auto-skips secrets
dotfiles install --unattended

# Or disable secrets in config
cat > config.yml <<EOF
secrets:
  provider: ""
EOF
```

### Go Not Found

**Problem**: Go binary not in PATH in user data scripts

**Solution**:
```bash
# Add Go to PATH
export PATH="/usr/local/go/bin:$PATH"
go build -o bin/dotfiles .
```

### Permission Denied

**Problem**: Running as wrong user

**Solution**:
```bash
# Terraform - run as target user
user_data = <<-EOF
  #!/bin/bash
  sudo -u ubuntu bash -c '
    cd ~
    git clone https://github.com/garygentry/dotfiles.git .dotfiles
    cd .dotfiles
    go build -o bin/dotfiles .
    ./bin/dotfiles install --unattended
  '
EOF
```

## Complete Examples

### AWS Auto Scaling Group

```hcl
resource "aws_launch_template" "dotfiles" {
  name_prefix   = "dotfiles-"
  image_id      = "ami-0c55b159cbfafe1f0"
  instance_type = "t3.micro"

  user_data = base64encode(templatefile("${path.module}/user-data.sh", {
    dotfiles_repo    = "https://github.com/garygentry/dotfiles.git"
    dotfiles_profile = "server"
    dotfiles_version = "main"
  }))

  tag_specifications {
    resource_type = "instance"
    tags = {
      Name = "dotfiles-server"
    }
  }
}

resource "aws_autoscaling_group" "dotfiles" {
  desired_capacity = 2
  max_size         = 4
  min_size         = 1

  launch_template {
    id      = aws_launch_template.dotfiles.id
    version = "$Latest"
  }

  vpc_zone_identifier = var.subnet_ids
}
```

**user-data.sh:**

```bash
#!/bin/bash
set -euo pipefail

# Install dependencies
apt-get update
apt-get install -y git golang-go curl

# Clone and install as ubuntu user
sudo -u ubuntu bash <<'SCRIPT'
cd ~
git clone ${dotfiles_repo} .dotfiles
cd .dotfiles
git checkout ${dotfiles_version}
go build -o bin/dotfiles .
./bin/dotfiles install --unattended --profile ${dotfiles_profile} --skip-failed

# Verify
./bin/dotfiles status
SCRIPT

echo "Dotfiles installation complete"
```

### GitLab CI

```yaml
# .gitlab-ci.yml
stages:
  - test
  - build

test-dotfiles:
  stage: test
  image: ubuntu:22.04
  before_script:
    - apt-get update && apt-get install -y git golang-go
  script:
    - go build -o bin/dotfiles .
    - ./bin/dotfiles install --unattended --dry-run
    - ./bin/dotfiles install --unattended --profile ci --skip-failed
    - ./bin/dotfiles status
  artifacts:
    when: on_failure
    paths:
      - .state/
    expire_in: 1 week

build-image:
  stage: build
  image: docker:latest
  services:
    - docker:dind
  script:
    - docker build -t myapp/dotfiles:latest .
    - docker run myapp/dotfiles:latest dotfiles status
```

## Summary

The `--unattended` flag makes the dotfiles system fully compatible with automated workflows:

- ✅ Zero interactive prompts
- ✅ Automatic fallback to defaults
- ✅ Works in CI/CD, IaC, and containers
- ✅ Compatible with all major cloud providers
- ✅ Comprehensive error handling
- ✅ Flexible profile system

For questions or issues, see the [main README](../README.md) or [Troubleshooting Guide](./troubleshooting.md).
