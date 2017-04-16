//Use only one containerTemplate otherwise it create an error "Pipe not connected" see https://issues.jenkins-ci.org/browse/JENKINS-40825
podTemplate(label: 'buildpod', containers: [
    containerTemplate(name: 'golang', image: 'kubermatic/golang', ttyEnabled: true, command: 'cat')
  ]) {
    node ('buildpod') {

        stage('Check'){
           sh 'echo $PWD'
           sh 'CGO_ENABLED=0 GOBUILD="go install" make install'
           sh 'CGO_ENABLED=0 GOBUILD="go install" make check'
        }
        stage('Test'){
           sh 'CGO_ENABLED=0 GOBUILD="go install" make test'
        }
        stage('Build'){
            sh 'CGO_ENABLED=0 GOBUILD="go install" make build'
            sh 'CGO_ENABLED=0 GOBUILD="go install" make docker'
        }
        stage('Push'){
            sh 'CGO_ENABLED=0 GOBUILD="go install" make push'
        }

        stage('Deploy'){
                echo "echo"
        }
    }
}