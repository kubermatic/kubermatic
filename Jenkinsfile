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
            try {
                notifyBuild('STARTED')

                stage('setup workdir'){
                    container('golang') {
                        sh("mkdir -p /go/src/github.com/kubermatic")
                        sh("ln -s `pwd` /go/src/github.com/kubermatic/api")
                        sh("cd /go/src/github.com/kubermatic/api")
                        checkout scm
                    }
                }
                stage('Check'){
                    container('golang') {
                       sh("cd /go/src/github.com/kubermatic/api && make install")
                       sh("cd /go/src/github.com/kubermatic/api && make check")
                    }
                }
                stage('Test'){
                    container('golang') {
                       sh("cd /go/src/github.com/kubermatic/api && make test")
                    }
                }
                stage('Build'){
                    container('golang') {
                        sh("cd /go/src/github.com/kubermatic/api && make build")
                        sh("cd /go/src/github.com/kubermatic/api && make docker-build")
                    }
                }
                stage('Push'){
                    container('golang') {
                        withCredentials([[$class: 'UsernamePasswordMultiBinding', credentialsId: 'docker',
                                usernameVariable: 'USERNAME', passwordVariable: 'PASSWORD']]) {
                            sh("docker login --username=${env.USERNAME} --password=${env.PASSWORD}")
                            sh("cd /go/src/github.com/kubermatic/api && make docker-push")
                        }
                    }
                }
                stage('Deploy'){
                        echo "echo"
                }
            } catch (e) {
               // If there was an exception thrown, the build failed
               currentBuild.result = "FAILED"
               throw e
             } finally {
               // Success or failure, always send notifications
               notifyBuild(currentBuild.result)
             }
        }
    }
}


def notifyBuild(String buildStatus = 'STARTED') {
  // build status of null means successful
  buildStatus =  buildStatus ?: 'SUCCESSFUL'

  // Default values
  def colorName = 'RED'
  def colorCode = '#FF0000'
  def subject = "${buildStatus}: Job '${env.JOB_NAME} [${env.BUILD_NUMBER}]'"
  def summary = "${subject} (${env.BUILD_URL})"
  def details = """<p>STARTED: Job '${env.JOB_NAME} [${env.BUILD_NUMBER}]':</p>
    <p>Check console output at &QUOT;<a href='${env.BUILD_URL}'>${env.JOB_NAME} [${env.BUILD_NUMBER}]</a>&QUOT;</p>"""

  // Override default values based on build status
  if (buildStatus == 'STARTED') {
    color = 'YELLOW'
    colorCode = '#FFFF00'
  } else if (buildStatus == 'SUCCESSFUL') {
    color = 'GREEN'
    colorCode = '#00FF00'
  } else {
    color = 'RED'
    colorCode = '#FF0000'
  }

  // Send notifications
  slackSend (color: colorCode, message: summary)

  emailext(
      subject: subject,
      body: details,
      recipientProviders: [[$class: 'DevelopersRecipientProvider']]
    )
}


