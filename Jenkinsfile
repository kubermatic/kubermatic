podTemplate(label: 'buildpod', containers: [
    containerTemplate(name: 'alpine', image: 'alpine:3.5', ttyEnabled: true, command: 'cat'),
    containerTemplate(name: 'golang', image: 'kubermatic/golang', ttyEnabled: true, command: 'cat')
  ]) {
    node ('buildpod') {

        stage 'Checkout'
        checkout scm

        stage 'Check'
               container('golang') {
                    sh 'CGO_ENABLED=0 GOBUILD="go install" make install'
               }
               container('golang') {
                    sh 'CGO_ENABLED=0 GOBUILD="go install" make check'
               }
        stage 'Test'
               container('golang') {
                    sh 'CGO_ENABLED=0 GOBUILD="go install" make test'
               }
        stage 'Build'
               container('golang') {
                    sh 'CGO_ENABLED=0 GOBUILD="go install" make build'
               }
               container('golang') {
                    sh 'CGO_ENABLED=0 GOBUILD="go install" make docker'
               }
        stage 'Push'
               container('golang') {
                    sh 'CGO_ENABLED=0 GOBUILD="go install" make push'
               }

        stage 'Deploy'
                echo "echo"

    }
}