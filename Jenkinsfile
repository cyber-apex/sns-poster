pipeline {
    agent any
    
    tools {
        go 'go-1.24'  // Make sure this matches your Jenkins Go tool configuration
    }
    
    environment {
        GO_VERSION = '1.24'
        PROJECT_NAME = 'xhs-poster'
        BINARY_NAME = 'xhs-poster'
        GOOS = 'linux'
        GOARCH = 'amd64'
        CGO_ENABLED = '0'
    }
    
    stages {
        stage('Checkout') {
            steps {
                echo 'Checking out source code...'
                checkout scm
            }
        }
        
        stage('Environment Setup') {
            steps {
                echo 'Setting up Go environment...'
                sh '''
                    go version
                    go env
                    echo "GOPATH: $GOPATH"
                    echo "GOROOT: $GOROOT"
                '''
            }
        }
        
        stage('Dependencies') {
            steps {
                echo 'Installing dependencies...'
                sh '''
                    go mod download
                    go mod verify
                    go mod tidy
                '''
            }
        }
        
        stage('Build') {
            parallel {
                stage('Linux Build') {
                    steps {
                        echo 'Building for Linux...'
                        sh '''
                            export GOOS=linux
                            export GOARCH=amd64
                            export CGO_ENABLED=0
                            
                            go build -v -o ${BINARY_NAME}-linux-amd64 .
                            
                            # Verify the binary
                            file ${BINARY_NAME}-linux-amd64
                            ls -la ${BINARY_NAME}-linux-amd64
                        '''
                    }
                }
            }
        }
    }
    
    post {
        always {
            echo 'Cleaning up...'
            
            // Archive build artifacts
            archiveArtifacts artifacts: 'release/*', fingerprint: true, allowEmptyArchive: true
            archiveArtifacts artifacts: '${BINARY_NAME}-*', fingerprint: true, allowEmptyArchive: true
            
            // Clean workspace
            sh '''
                # Kill any remaining processes
                pkill -f xhs-poster || true
                
                # Clean build artifacts from workspace (but keep archived ones)
                rm -f ${BINARY_NAME}-*
                rm -rf release/
            '''
        }
        
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
