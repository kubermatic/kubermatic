//Use only one containerTemplate otherwise it create an error "Pipe not connected" see https://issues.jenkins-ci.org/browse/JENKINS-40825
podTemplate(label: 'buildpod', containers: [
    containerTemplate(name: 'golang', image: 'kubermatic/golang', ttyEnabled: true, command: 'cat')
  ]) {
    node ('buildpod') {
      withEnv([
        "CGO_ENABLED=0",
        "GOBUILD='go install'"
      ]) {

            stage('setup workdir'){
                container('golang') {
                    sh 'mkdir -p /go/src/github.com/kubermatic'
                    sh 'ln -s `pwd` /go/src/github.com/kubermatic/api'
                    sh 'cd /go/src/github.com/kubermatic/api'
                }
            }
            stage('Check'){
                container('golang') {
                   sh 'make install'
                   sh 'make check'
                }
            }
            stage('Test'){
                container('golang') {
                   sh 'make test'
                }
            }
            stage('Build'){
                container('golang') {
                    sh 'make build'
                    sh 'make docker'
                }
            }
            stage('Push'){
                container('golang') {
                    sh 'make push'
                }
            }
            stage('Deploy'){
                    echo "echo"
            }
        }
    }
}

