pipeline {
    agent any
    
    environment {
        PROJECT_NAME = 'xhs-poster'
        BINARY_NAME = 'xhs-poster'
        GOPATH="/usr/local/go/bin:/var/lib/jenkins/go/bin"
    }
    
    stages {
        stage('Checkout') {
            steps {
                echo 'Checking out source code...'
                checkout scm
            }
        }
        
        stage('Dependencies') {
            steps {
                echo 'Installing dependencies...'
                sh '''
                    PATH=$PATH:${GOPATH}
                    go get
                '''
            }
        }
        
        stage('Build') {
            parallel {
                stage('Linux Build') {
                    steps {
                        echo 'Building for Linux...'
                        sh '''
                            PATH=$PATH:${GOPATH}
                            export GOOS=linux
                            export GOARCH=amd64
                            export CGO_ENABLED=0

                            go version
                            go env
                            
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
