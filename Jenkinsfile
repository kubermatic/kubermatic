//Use only one containerTemplate otherwise it create an error "Pipe not connected" see https://issues.jenkins-ci.org/browse/JENKINS-40825
podTemplate(label: 'buildpod', containers: [
    containerTemplate(name: 'golang',
                      image: 'kubermatic/golang:test',
                      ttyEnabled: true,
                      command: 'cat',
                      envVars: [
                            containerEnvVar(key: 'DOCKER_HOST', value: 'tcp://localhost:2375'),
                      ]
    ),
    containerTemplate(name: 'docker',
                      image: 'docker:stable-dind',
                      ttyEnabled: true,
                      privileged: true,
                      command: 'dockerd-entrypoint.sh')
  ],
  volumes: [
      emptyDirVolume(mountPath: '/var/lib/docker', memory: false)
  ],
  ) {
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
                    sh 'cd /go/src/github.com/kubermatic/api && make build'
                    sh 'cd /go/src/github.com/kubermatic/api && make docker'
                }
            }
            stage('Push'){
                container('golang') {
                    sh 'cd /go/src/github.com/kubermatic/api && make push'
                }
            }
            stage('Deploy'){
                    echo "echo"
            }
        }
    }
}