pipeline {
    agent any
    
    environment {
        PROJECT_NAME = 'sns-poster'
        BINARY_NAME = 'sns-poster'
        SERVICE_NAME = 'sns-poster.service'
        // Use workspace subdirectory for Go modules and cache
        GOCACHE = "${env.WORKSPACE}/.gocache"
        GOMODCACHE = "${env.WORKSPACE}/.gomodcache"
    }
    
    stages {
        stage('Checkout') {
            steps {
                echo 'Checking out source code...'
                checkout scm
            }
        }
        
        stage('Setup Go Environment') {
            steps {
                echo 'Setting up Go environment...'
                sh '''
                    # Add Go to PATH
                    export PATH=/usr/local/go/bin:$PATH
                    
                    # Create cache directories in workspace
                    mkdir -p ${WORKSPACE}/.gocache
                    mkdir -p ${WORKSPACE}/.gomodcache
                    
                    # Set Go environment variables
                    export GOCACHE=${WORKSPACE}/.gocache
                    export GOMODCACHE=${WORKSPACE}/.gomodcache
                    export CGO_ENABLED=0
                    
                    # Verify Go installation
                    go version
                    go env GOCACHE
                    go env GOMODCACHE
                    
                    # Download dependencies
                    go mod download
                    go mod verify
                '''
            }
        }
        
        stage('Build') {
            steps {
                echo 'Building SNS Poster for Linux AMD64...'
                sh '''
                    # Add Go to PATH
                    export PATH=/usr/local/go/bin:$PATH
                    
                    # Set Go environment variables
                    export GOCACHE=${WORKSPACE}/.gocache
                    export GOMODCACHE=${WORKSPACE}/.gomodcache
                    export GOOS=linux
                    export GOARCH=amd64
                    export CGO_ENABLED=0

                    echo "Go version:"
                    go version
                    
                    echo "Building binary..."
                    mkdir -p bin
                    go build -ldflags="-s -w" -o bin/${BINARY_NAME}-linux-amd64 ./cmd/sns-poster
                    
                    echo "Verifying binary:"
                    file bin/${BINARY_NAME}-linux-amd64
                    ls -la bin/${BINARY_NAME}-linux-amd64
                    
                    echo "Binary size: $(du -h bin/${BINARY_NAME}-linux-amd64 | cut -f1)"
                '''
            }
        }
        
        stage('Test') {
            steps {
                echo 'Running tests...'
                sh '''
                    # Add Go to PATH
                    export PATH=/usr/local/go/bin:$PATH
                    
                    # Set Go environment variables
                    export GOCACHE=${WORKSPACE}/.gocache
                    export GOMODCACHE=${WORKSPACE}/.gomodcache
                    export CGO_ENABLED=0
                    
                    echo "Running Go tests..."
                    go test -v ./...
                    
                    echo "Checking Go modules..."
                    go mod tidy
                    go mod verify
                    
                    echo "Running Go vet..."
                    go vet ./...
                '''
            }
        }

        stage('Deploy') {
            steps {
                echo 'Deploying SNS Poster...'
                sh '''
                    echo "Stopping existing service..."
                    sudo systemctl stop ${SERVICE_NAME} || true
                    
                    echo "Creating deployment directories..."
                    sudo mkdir -p /opt/sns-poster
                    sudo mkdir -p /var/logs/sns-poster
                    
                    echo "Copying binary to deployment location..."
                    sudo cp bin/${BINARY_NAME}-linux-amd64 /opt/sns-poster/${BINARY_NAME}
                    sudo chmod +x /opt/sns-poster/${BINARY_NAME}
                    sudo chown root:root /opt/sns-poster/${BINARY_NAME}
                    
                    echo "Installing service file..."
                    if [ -f scripts/sns-poster.service ]; then
                        sudo cp scripts/sns-poster.service /etc/systemd/system/
                        sudo systemctl daemon-reload
                    else
                        echo "Warning: Service file not found at scripts/sns-poster.service"
                    fi
                    
                    echo "Starting service..."
                    sudo systemctl start ${SERVICE_NAME}
                '''
            }
        }

        stage('Check') {
            steps {
                echo 'Checking service status...'
                sh '''
                    sudo systemctl status ${SERVICE_NAME}
                    
                    sleep 10
                    
                    echo "Testing health endpoint..."
                    curl -f http://localhost:6170/health || (echo "Health check failed" && exit 1)
                '''
            }
        }
    }
    
    post {
        success {
            echo '✅ SNS Poster Pipeline completed successfully!'
            
            // Archive build artifacts
            archiveArtifacts artifacts: 'bin/sns-poster-linux-amd64', fingerprint: true
            
            // Notify on success (configure as needed)
            script {
                if (env.BRANCH_NAME == 'main' || env.BRANCH_NAME == 'master') {
                    // Add notification logic here (Slack, email, etc.)
                    echo '🚀 SNS Poster main branch build succeeded - deployed and ready!'
                    echo "📦 Binary: sns-notify-linux-amd64"
                    echo "🌐 Service: http://localhost:6170"
                    echo "🏥 Health: http://localhost:6170/health"
                }
            }
        }
        
        failure {
            echo '❌ SNS Poster Pipeline failed!'
            
            // Notify on failure
            script {
                // Add notification logic here
                echo '🔥 SNS Poster build failed - please check the logs'
                echo '📋 Common issues:'
                echo '  • Go dependencies not available'
                echo '  • Module build errors'
                echo '  • Service deployment issues'
            }
        }
        
      
        always {
            echo '🧹 Cleaning up...'
            
            // Clean up workspace to save disk space
            script {
                echo "Workspace size before cleanup: \$(du -sh \$WORKSPACE | cut -f1)"
                
                // Clean Go cache if it gets too large
                sh '''
                    if [ -d "${WORKSPACE}/.gocache" ]; then
                        find ${WORKSPACE}/.gocache -type f -atime +7 -delete 2>/dev/null || true
                    fi
                    if [ -d "${WORKSPACE}/.gomodcache" ]; then
                        find ${WORKSPACE}/.gomodcache -type f -atime +7 -delete 2>/dev/null || true
                    fi
                '''
                
                echo "Cleanup completed"
            }
        }
    }
}
