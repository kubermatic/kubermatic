//Use only one containerTemplate otherwise it create an error "Pipe not connected" see https://issues.jenkins-ci.org/browse/JENKINS-40825
podTemplate(label: 'buildpod', containers: [
    containerTemplate(name: 'golang',
                      image: 'kubermatic/golang:test',
                      alwaysPullImage: true,
                      ttyEnabled: true,
                      command: 'cat'
  )],
  volumes: [
        hostPathVolume(hostPath: '/var/run/docker.sock', mountPath: '/var/run/docker.sock'),
        hostPathVolume(hostPath: '/usr/bin/docker', mountPath: '/usr/bin/docker')
  ])
  {
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
                    checkout scm
                }
            }
            stage('Check'){
                container('golang') {
                   sh 'cd /go/src/github.com/kubermatic/api && make install'
                   sh 'cd /go/src/github.com/kubermatic/api && make check'
                }
            }
            stage('Test'){
                container('golang') {
                   sh 'cd /go/src/github.com/kubermatic/api && make test'
                }
            }
            stage('Build'){
                container('golang') {
                    sh 'cd /go/src/github.com/kubermatic/api && make docker-build'
                }
            }
            stage('Push'){
                container('golang') {
                    sh 'cd /go/src/github.com/kubermatic/api && make docker-push'
                }
            }
            stage('Deploy'){
                    echo "echo"
            }
        }
    }
}