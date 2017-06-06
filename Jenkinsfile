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
                // Setting source code related global variable once so it can be reused.
                def gitCommit = getRevision()
                env.GIT_SHA = gitCommit
                env.GIT_COMMIT = gitCommit.take(7)
                env.GIT_TAG = getTag()

                def tags
                def stage
                if (env.BRANCH_NAME == "develop" && env.GIT_TAG !=  null) {
                    tags = "${env.GIT_TAG} latest"
                    stage_system = "prod"
                } else if (env.BRANCH_NAME == "develop") {
                    tags = "${env.GIT_TAG} latest"
                    stage_system = "staging"
                } else {
                    tags = "${env.BRANCH_NAME} dev"
                    stage_system = "dev"
                }
                buildPipeline(tags, stage_system)
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


def buildPipeline(tags, stage_system) {

    stage('Install deps'){
        container('golang') {
           sh("cd /go/src/github.com/kubermatic/api && make install")
        }
    }
    stage('Check'){
        container('golang') {
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
            sh("cd /go/src/github.com/kubermatic/api && make TAG='${tags}' docker-build")
        }
    }
    stage('Push'){
        container('golang') {
            withCredentials([usernamePassword(credentialsId: 'docker',
                    passwordVariable: 'PASSWORD', usernameVariable: 'USERNAME')]) {
                sh("docker login --username=$USERNAME --password=$PASSWORD")
                sh("cd /go/src/github.com/kubermatic/api && make TAG='${tags}' docker-push")
            }
        }
    }
    switch (stage_system) {
         case "staging":
                stage('Deploy Staging'){
                    container('golang') {
                        withCredentials([string(credentialsId: 'kubeconfig-dev', variable: 'KUBECONFIG')]) {
                            sh("echo '$KUBECONFIG'>kubeconfig && kubectl --kubeconfig=kubeconfig set image deployment/kubermatic-api-v1 api=kubermatic/api:'${tags}' --namespace=kubermatic")
                            sh("echo '$KUBECONFIG'>kubeconfig && kubectl --kubeconfig=kubeconfig set image deployment/cluster-controller-v1 cluster-controller=kubermatic/api:'${tags}' --namespace=kubermatic")
                        }

                    }
                }
         default:
                stage('Deploy Dev'){
                    container('golang') {
                        withCredentials([string(credentialsId: 'kubeconfig-dev', variable: 'KUBECONFIG')]) {
                            sh("echo '$KUBECONFIG'>kubeconfig && kubectl --kubeconfig=kubeconfig set image deployment/kubermatic-api-v1 api=kubermatic/api:'${tags}' --namespace=kubermatic")
                            sh("echo '$KUBECONFIG'>kubeconfig && kubectl --kubeconfig=kubeconfig set image deployment/cluster-controller-v1 cluster-controller=kubermatic/api:'${tags}' --namespace=kubermatic")
                        }

                    }
                }
    }
}

// Finds the revision from the source code.
def getRevision() {
  // Code needs to be checked out for this.
  return sh(returnStdout: true, script: 'git rev-parse --verify HEAD').trim()
}

def getTag() {
  return sh(returnStdout: true, script: 'git describe 2> /dev/null || exit 0').trim()
}

def notifyBuild(String buildStatus = 'STARTED') {
  // build status of null means successful
  buildStatus =  buildStatus ?: 'SUCCESSFUL'

  // Default values
  def colorName = 'RED'
  def colorCode = '#FF0000'
  def msg = "${buildStatus}: ${env.JOB_NAME} #${env.BUILD_NUMBER}"


  // Override default values based on build status
  if (buildStatus == 'STARTED') {
    color = 'YELLOW'
    colorCode = '#FFFF00'
  } else if (buildStatus == 'SUCCESSFUL') {
    color = 'GREEN'
    colorCode = '#00FF00'
  }

  // Send notifications
  slackSend (color: colorCode, message: msg)

}



