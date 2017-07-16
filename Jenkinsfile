//Use only one containerTemplate otherwise it create an error "Pipe not connected" see https://issues.jenkins-ci.org/browse/JENKINS-40825
podTemplate(label: 'api', containers: [
    containerTemplate(name: 'golang', image: 'kubermatic/golang:test', ttyEnabled: true, command: 'cat'),
    containerTemplate(name: 'docker', image: 'kubermatic/docker:1.11', ttyEnabled: true, command: 'cat')
  ],
    volumes: [hostPathVolume(hostPath: '/var/run/docker.sock', mountPath: '/var/run/docker.sock'),
    emptyDirVolume(mountPath: '/go/src/github.com/kubermatic', memory: false)]
  )
  {
    node ('api') {
        try {
            notifyBuild('STARTED')

            stage('setup workdir'){
                container('golang') {
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
            env.DOCKER_TAG = env.BRANCH_NAME.replaceAll("/","_")

            if (env.BRANCH_NAME == "develop" && env.GIT_TAG !=  "") {
                dockerTags = "${env.GIT_TAG} latest"
                deployTag  = "${env.GIT_TAG}"
                stageSystem = "prod"
            } else if (env.BRANCH_NAME == "develop") {
                dockerTags = "${env.DOCKER_TAG} staging"
                deployTag  = "${env.DOCKER_TAG}"
                stageSystem = "staging"
            } else {
                dockerTags = "${env.DOCKER_TAG} dev"
                deployTag  = "${env.DOCKER_TAG}"
                stageSystem = "dev"
            }
            buildPipeline(dockerTags,deployTag, stageSystem)
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

def buildPipeline(dockerTags, deployTag, stageSystem) {

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
    stage('Build go'){
        container('golang') {
            sh("cd /go/src/github.com/kubermatic/api && CGO_ENABLED=0 make build")
        }
    }
    stage('Build docker image'){
        container('docker') {
            sh("cd /go/src/github.com/kubermatic/api && make TAGS='${dockerTags}' docker-build")
        }
    }
    stage('Push'){
        container('docker') {
            withCredentials([usernamePassword(credentialsId: 'docker',
                    passwordVariable: 'PASSWORD', usernameVariable: 'USERNAME')]) {
                sh("docker login --username=$USERNAME --password=$PASSWORD")
                sh("cd /go/src/github.com/kubermatic/api && make TAGS='${dockerTags}' docker-push")
            }
        }
    }
    switch (stageSystem) {
         case "staging":
                stage('Deploy Staging'){
                    container('docker') {
                        withCredentials([string(credentialsId: 'kubeconfig-dev', variable: 'KUBECONFIG')]) {
                            sh("echo '$KUBECONFIG'>kubeconfig && kubectl --kubeconfig=kubeconfig set image deployment/kubermatic-api-v1 api=kubermatic/api:'${deployTag}' --namespace=kubermatic")
                            sh("echo '$KUBECONFIG'>kubeconfig && kubectl --kubeconfig=kubeconfig set image deployment/cluster-controller-v1 cluster-controller=kubermatic/api:'${deployTag}' --namespace=kubermatic")
                        }

                    }
                }
         default:
                stage('Deploy Dev'){
                    container('docker') {
                        withCredentials([string(credentialsId: 'kubeconfig-dev', variable: 'KUBECONFIG')]) {
                            sh("echo '$KUBECONFIG'>kubeconfig && kubectl --kubeconfig=kubeconfig set image deployment/kubermatic-api-v1 api=kubermatic/api:'${deployTag}' --namespace=kubermatic")
                            sh("echo '$KUBECONFIG'>kubeconfig && kubectl --kubeconfig=kubeconfig set image deployment/cluster-controller-v1 cluster-controller=kubermatic/api:'${deployTag}' --namespace=kubermatic")
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
  def msg = "${buildStatus}: ${env.BUILD_URL}display/redirect"

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
