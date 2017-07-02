//Use only one containerTemplate otherwise it create an error "Pipe not connected" see https://issues.jenkins-ci.org/browse/JENKINS-40825
podTemplate(label: 'buildpod', containers: [
  containerTemplate(name: 'golang',
   image: 'realfake/golang:test',
   alwaysPullImage: true,
   ttyEnabled: true,
   command: 'cat'
  )
 ],
 volumes: [
  hostPathVolume(hostPath: '/var/run/docker.sock', mountPath: '/var/run/docker.sock'),
  hostPathVolume(hostPath: '/usr/bin/docker', mountPath: '/usr/bin/docker')
 ]) {
 node('buildpod') {
  withEnv([
   "GOBUILD='go build'"
  ]) {
   try {
    notifyBuild('STARTED')

    stage('setup workdir') {
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
    tags = "${env.GIT_COMMIT} "

    if (env.BRANCH_NAME == "develop") {
     tags += "latest"
     stage_system = "staging"
    } else if (env.BRANCH_NAME == "master") {
     tags += "stable"
     stage_system = "prod"
    } else {
     tags += "dev"
     stage_system = "dev"
    }

    buildPipeline(env.GIT_COMMIT, tags, stage_system)
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


def buildPipeline(gittag, tags, stage_system) {
 stage('Install deps') {
  container('golang') {
   sh """
   cd /go/src/github.com/kubermatic/api && make install
   """
  }
 }
 stage('Check') {
  container('golang') {
   sh """
   cd /go/src/github.com/kubermatic/api && make check
   """
  }
 }
 stage('Test') {
  container('golang') {
   sh """
   cd /go/src/github.com/kubermatic/api && make test
   """
  }
 }
 stage('Build') {
  container('golang') {
   sh """
   cd /go/src/github.com/kubermatic/api && make TAGS=\"${tags}\" docker-build
   """
  }
 }
 stage('Push') {
  container('golang') {
   withCredentials([usernamePassword(credentialsId: 'docker',
    passwordVariable: 'PASSWORD', usernameVariable: 'USERNAME')]) {
    sh """
    docker login --username=$USERNAME --password=$PASSWORD
    cd /go/src/github.com/kubermatic/api && make TAGS=\"${tags}\" docker-push
    """
   }
  }
 }

 // Stage system is either prod/staging/dev (no user input!)
 String credentials = "kubeconfig-" + stage_system
 stage('Deploy dev') {
  container('golang') {
   withCredentials([string(credentialsId: credentials, variable: 'KUBECONFIG')]) {
    sh """
    echo '$KUBECONFIG'>kubeconfig
    kubectl --kubeconfig=kubeconfig set image deployment/kubermatic-api-v1 api=kubermatic/api:'${gittag}' --namespace=kubermatic
    kubectl --kubeconfig=kubeconfig set image deployment/cluster-controller-v1 cluster-controller=kubermatic/api:'${gittag}' --namespace=kubermatic
    """
   }
  }
 }

 // Don't run end-2-end tests on every feature commit
 // TODO(realfake): Implement a commit trigger manually to e2e test
 if (stage_system != "dev") {
  stage('End-to-End') {
   container('golang') {
    withCredentials([usernamePassword(credentialsId: 'docker',
     passwordVariable: 'PASSWORD', usernameVariable: 'USERNAME')]) {
     sh """
     docker login --username=$USERNAME --password=$PASSWORD
     cd /go/src/github.com/kubermatic/api
     make client-up
     make e2e
     make client-down
     """
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
 return sh(returnStdout: true, script: 'git describe --tags --always || exit 0').trim()
}

def notifyBuild(String buildStatus = 'STARTED') {
 // build status of null means successful
 buildStatus = buildStatus ? : 'SUCCESSFUL'

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
 slackSend(color: colorCode, message: msg)

}