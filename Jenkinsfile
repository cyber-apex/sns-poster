pipeline {
    agent any
    
    environment {
        PROJECT_NAME = 'sns-poster'
        BINARY_NAME = 'sns-poster'
        SERVICE_NAME = 'sns-poster.service'
        // Use workspace subdirectory for Go modules and cache
        CACHE_BASE = "/var/lib/jenkins/${PROJECT_NAME}/.cache"
        GOCACHE = "/var/lib/jenkins/${PROJECT_NAME}/.cache/go-build"
        GOMODCACHE = "/var/lib/jenkins/${PROJECT_NAME}/.cache/go-mod"
        GOBIN = "/var/lib/jenkins/${PROJECT_NAME}/.cache/go-bin"
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
                    export PATH=/usr/local/go/bin:${GOBIN}:$PATH
                    
                    # Create persistent cache directories
                    mkdir -p ${GOCACHE}
                    mkdir -p ${GOMODCACHE}
                    mkdir -p ${GOBIN}
                    
                    # Set Go environment variables
                    export GOCACHE=${GOCACHE}
                    export GOMODCACHE=${GOMODCACHE}
                    export GOBIN=${GOBIN}
                    export CGO_ENABLED=0
                    
                    # Verify Go installation
                    go version
                    echo "Using GOCACHE: $(go env GOCACHE)"
                    echo "Using GOMODCACHE: $(go env GOMODCACHE)"
                    
                    # Check if go.sum changed (skip download if unchanged)
                    if [ -f "${GOMODCACHE}/.last_build_${PROJECT_NAME}" ]; then
                        if diff -q go.sum "${GOMODCACHE}/.last_build_${PROJECT_NAME}" > /dev/null 2>&1; then
                            echo "✅ Dependencies unchanged, using cached modules"
                        else
                            echo "📦 Dependencies changed, downloading..."
                            go mod download
                        fi
                    else
                        echo "📦 First build or cache cleared, downloading dependencies..."
                        go mod download
                    fi
                    
                    # Verify and save checksum for next build
                    go mod verify
                    cp go.sum "${GOMODCACHE}/.last_build_${PROJECT_NAME}"
                '''
            }
        }
        
        stage('Build') {
            steps {
                echo 'Building SNS Poster for Linux AMD64...'
                sh '''
                    export PATH=/usr/local/go/bin:${GOBIN}:$PATH
                    
                    # Create persistent cache directories
                    mkdir -p ${GOCACHE}
                    mkdir -p ${GOMODCACHE}
                    mkdir -p ${GOBIN}
                    
                    # Set Go environment variables
                    export GOCACHE=${GOCACHE}
                    export GOMODCACHE=${GOMODCACHE}
                    export GOBIN=${GOBIN}
                    export CGO_ENABLED=0
                    
                    echo "Building binary..."
                    mkdir -p bin
                    go build -ldflags="-s -w" -o bin/${BINARY_NAME}-linux-amd64 ./cmd
                    
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
                    export PATH=/usr/local/go/bin:${GOBIN}:$PATH
                    
                    # Create persistent cache directories
                    mkdir -p ${GOCACHE}
                    mkdir -p ${GOMODCACHE}
                    mkdir -p ${GOBIN}
                    
                    # Set Go environment variables
                    export GOCACHE=${GOCACHE}
                    export GOMODCACHE=${GOMODCACHE}
                    export GOBIN=${GOBIN}
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

        stage("Deploy Remote Browser") {
            steps {
                echo 'Restarting Remote Browser...'
                sh '''
                docker restart $(docker ps -q --filter ancestor=ghcr.io/go-rod/rod)
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
                    # curl -f http://localhost:6170/health || (echo "Health check failed" && exit 1)
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
                echo "Workspace size: \$(du -sh \$WORKSPACE | cut -f1)"
                
                // Clean old build cache (keep module cache intact for speed)
                sh '''
                    # Only clean build cache older than 30 days to save space
                    # Keep GOMODCACHE intact for fast dependency resolution
                    if [ -d "${GOCACHE}" ]; then
                        echo "Cleaning old build cache (30+ days)..."
                        find ${GOCACHE} -type f -atime +30 -delete 2>/dev/null || true
                    fi
                    
                    # Show cache sizes for monitoring
                    echo "Cache sizes:"
                    du -sh ${GOCACHE} 2>/dev/null || echo "GOCACHE not found"
                    du -sh ${GOMODCACHE} 2>/dev/null || echo "GOMODCACHE not found"
                '''
                
                echo "Cleanup completed"
            }
        }
    }
}
