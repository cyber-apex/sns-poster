# Jenkins CI/CD Setup for SNS Notify

This document explains how to set up Jenkins CI/CD for the SNS Notify project.

## Prerequisites

### Jenkins Requirements
- Jenkins 2.400+ with Pipeline plugin
- Go plugin for Jenkins
- Blue Ocean plugin (recommended for better UI)

### Required Jenkins Tools
Configure these tools in Jenkins Global Tool Configuration:

1. **Go Installation**
   - Name: `go-1.24`
   - Version: Go 1.24 or later
   - Installation method: Install from golang.org

### Required Jenkins Plugins
- Pipeline
- Git
- Go
- HTML Publisher (for coverage reports)
- Blue Ocean (optional, for better UI)

## Setup Instructions

### 1. Create Jenkins Job

1. **New Item** → **Multibranch Pipeline**
2. **Branch Sources** → **Git**
3. **Repository URL**: Your git repository URL
4. **Credentials**: Configure if needed
5. **Build Configuration** → **by Jenkinsfile**
6. **Script Path**: `Jenkinsfile`

### 2. Configure Webhooks (Optional)

For automatic builds on git push:

1. **GitHub**: Add webhook URL: `http://your-jenkins-url/github-webhook/`
2. **GitLab**: Add webhook URL: `http://your-jenkins-url/gitlab-webhook/`

### 3. Environment Variables

Configure these in Jenkins if needed:

```bash
# Optional environment variables
SLACK_WEBHOOK_URL=your-slack-webhook-url
EMAIL_RECIPIENTS=team@company.com
```

## Pipeline Stages

The Jenkins pipeline includes these stages:

### 🔍 **Code Quality**
- **Lint**: Code style checking with golangci-lint
- **Format**: Go code formatting verification
- **Vet**: Static analysis with go vet

### 🧪 **Testing**
- **Unit Tests**: Run with race detection and coverage
- **Integration Tests**: Health check and API testing
- **Security Scan**: Code security analysis with gosec

### 🏗️ **Build**
- **Multi-platform**: Linux, Windows, macOS
- **Binary Generation**: Statically linked binaries
- **Verification**: File type and size validation

### 📦 **Package**
- **Archive Creation**: tar.gz for Linux/macOS, zip for Windows
- **Artifact Storage**: Binaries and packages archived

### 🚀 **Integration**
- **Server Testing**: Start server and run health checks
- **API Testing**: Execute quick test scripts

## Pipeline Features

### ✅ **Parallel Execution**
- Code quality checks run in parallel
- Multi-platform builds execute simultaneously
- Reduces overall pipeline time

### 📊 **Reporting**
- **Coverage Reports**: HTML coverage reports published
- **Security Reports**: gosec security scan results
- **Artifact Archive**: All binaries and packages stored

### 🔔 **Notifications**
- Build status updates
- Success/failure notifications
- Configurable via Slack, email, or GitHub status

## Build Artifacts

After successful builds, these artifacts are available:

```
📁 Archived Artifacts
├── sns-notify-linux-amd64           # Linux binary
├── sns-notify-windows-amd64.exe     # Windows binary
├── sns-notify-darwin-amd64          # macOS binary
├── release/
│   ├── sns-notify-linux-amd64.tar.gz
│   ├── sns-notify-windows-amd64.zip
│   └── sns-notify-darwin-amd64.tar.gz
├── coverage.html                    # Coverage report
└── gosec-report.json               # Security scan report
```

## Pipeline Configuration

### Branch Strategy
- **Main/Master**: Full pipeline + integration tests
- **Develop**: Full pipeline
- **Feature branches**: Build and test only

### Quality Gates
- **Code formatting**: Must pass (fails build)
- **Go vet**: Must pass (fails build)
- **Linting**: Warning only (unstable build)
- **Security scan**: Warning only (unstable build)

## Customization

### Modify Pipeline Behavior

Edit `Jenkinsfile` to customize:
- Add/remove build stages
- Change quality gate thresholds
- Modify notification settings
- Add deployment stages

### Configuration File

Edit `.jenkins.yml` to adjust:
- Build targets and timeouts
- Testing configuration
- Quality thresholds
- Deployment settings

## Troubleshooting

### Common Issues

1. **Go tool not found**
   - Verify Go installation in Jenkins Global Tools
   - Check tool name matches `go-1.24`

2. **Permission denied on binaries**
   - Ensure Jenkins has execute permissions
   - Check file system permissions

3. **Integration tests timeout**
   - Increase timeout in pipeline
   - Check server startup requirements

4. **Coverage report not published**
   - Verify HTML Publisher plugin is installed
   - Check coverage.html file generation

### Debug Steps

1. Check Jenkins console output
2. Verify tool installations
3. Test commands manually on Jenkins agent
4. Review pipeline logs in Blue Ocean

## Security Considerations

- **Credentials**: Store sensitive data in Jenkins credentials
- **Secrets**: Never commit API keys or passwords
- **Permissions**: Limit Jenkins job permissions
- **Network**: Restrict Jenkins agent network access

## Maintenance

### Regular Tasks
- Update Go version in pipeline
- Review and update dependencies
- Monitor build performance
- Clean up old artifacts

### Performance Optimization
- Use Jenkins build cache
- Parallelize more stages
- Optimize Docker images (if using)
- Monitor resource usage

---

For more information, see:
- [Jenkins Pipeline Documentation](https://www.jenkins.io/doc/book/pipeline/)
- [Go Plugin Documentation](https://plugins.jenkins.io/golang/)
- [Blue Ocean Documentation](https://www.jenkins.io/doc/book/blueocean/)
