// Jenkinsfile for toolbridge-api
// Go-based API server CI pipeline

@Library('homelab') _

pipeline {
    agent {
        kubernetes {
            yaml homelab.podTemplate('golang')
        }
    }

    options {
        buildDiscarder(logRotator(numToKeepStr: '10'))
        timeout(time: 10, unit: 'MINUTES')
        disableConcurrentBuilds()
    }

    stages {
        stage('Build') {
            steps {
                container('golang') {
                    sh '''
                        echo "=== Building toolbridge-api ==="
                        go build -v ./...
                    '''
                }
            }
        }

        stage('Test') {
            steps {
                container('golang') {
                    sh '''
                        echo "=== Running tests ==="
                        go test -v ./...
                    '''
                }
            }
        }

        stage('Vet') {
            steps {
                container('golang') {
                    sh '''
                        echo "=== Running go vet ==="
                        go vet ./...
                    '''
                }
            }
        }
    }

    post {
        success {
            script {
                homelab.githubStatus('SUCCESS', 'Build passed')
            }
        }
        failure {
            script {
                homelab.githubStatus('FAILURE', 'Build failed')
                homelab.postFailurePrComment([repo: 'erauner/toolbridge-api'])
                homelab.notifyDiscordFailure()
            }
        }
    }
}
