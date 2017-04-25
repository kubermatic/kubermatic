node {
        stage("Main build") {

            checkout scm

            docker.image('kubermatic/golang:test').inside {

                stage('setup workdir'){
                    sh 'mkdir -p /go/src/github.com/kubermatic'
                    sh 'ln -s `pwd` /go/src/github.com/kubermatic/api'
                    sh 'cd /go/src/github.com/kubermatic/api'
                }

                stage('Check'){
                    sh 'cd /go/src/github.com/kubermatic/api && make install'
                    sh 'cd /go/src/github.com/kubermatic/api && make check'
                }


                stage('Test'){
                    sh 'cd /go/src/github.com/kubermatic/api && make test'
                }

                stage('Build'){
                    sh 'cd /go/src/github.com/kubermatic/api && make build'
                    sh 'cd /go/src/github.com/kubermatic/api && make docker'
                }

                stage('Push'){
                    sh 'cd /go/src/github.com/kubermatic/api && make push'
                }

                stage('Deploy'){
                    echo "echo"
                }

           }

        }

        // Clean up workspace
        step([$class: 'WsCleanup'])

}
