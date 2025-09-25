pipeline {
    agent any
    
    environment {
        PROJECT_NAME = 'xhs-poster'
        BINARY_NAME = 'xhs-poster'
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
                echo 'Building for Linux...'
                sh '''
                    # Add Go to PATH
                    export PATH=/usr/local/go/bin:$PATH
                    
                    # Set Go environment variables
                    export GOCACHE=${WORKSPACE}/.gocache
                    export GOMODCACHE=${WORKSPACE}/.gomodcache
                    export GOOS=linux
                    export GOARCH=amd64
                    export CGO_ENABLED=0

                    go version
                    go env
                    
                    go build -o ${BINARY_NAME}-linux-amd64 .
                    
                    # Verify the binary
                    file ${BINARY_NAME}-linux-amd64
                    ls -la ${BINARY_NAME}-linux-amd64
                '''
            }
        }
    }
    
    post {
        success {
            echo '✅ Pipeline completed successfully!'
            
            // Notify on success (configure as needed)
            script {
                if (env.BRANCH_NAME == 'main' || env.BRANCH_NAME == 'master') {
                    // Add notification logic here (Slack, email, etc.)
                    echo '🚀 Main branch build succeeded - ready for deployment!'
                }
            }
        }
        
        failure {
            echo '❌ Pipeline failed!'
            
            // Notify on failure
            script {
                // Add notification logic here
                echo '🔥 Build failed - please check the logs'
            }
        }
        
        unstable {
            echo '⚠️ Pipeline completed with warnings!'
            
            // Notify on unstable build
            script {
                echo '⚠️ Build completed with warnings - please review'
            }
        }
    }
}
