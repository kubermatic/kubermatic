def pipeline = new io.kubermatic.pipeline()
goBuildNode(pipeline){

         def goPath = "/go/src/github.com/kubermatic"
         def goImportPath = "/go/src/github.com/kubermatic/api"
         pipeline.setup ("golang", goPath, goImportPath)
         pipeline.setupENV()
         stage('Install deps'){
             container('golang') {
                sh("cd ${goImportPath} && make install")
             }
         }
         stage('Check'){
             container('golang') {
                sh("cd ${goImportPath} && make check")
             }
         }
         stage('Test'){
             container('golang') {
                sh("cd ${goImportPath} && make test")
             }
         }
         stage('Build go'){
             container('golang') {
                 sh("cd ${goImportPath} && CGO_ENABLED=0 make build")
             }
         }

        if (env.BRANCH_NAME == "develop" && env.GIT_TAG !=  "") {
            pipeline.dockerBuild("docker", "${env.DOCKER_TAG} latest" )
            pipeline.deploy("docker", "prod", "kubermatic", "deployment/kubermatic-api-v1", "api=kubermatic/api:${env.DOCKER_TAG}")
            pipeline.deploy("docker", "prod", "kubermatic", "deployment/cluster-controller-v1", "cluster-controller=kubermatic/api:${env.DOCKER_TAG}")
        } else if (env.BRANCH_NAME == "develop") {
            pipeline.dockerBuild("docker", "${env.DOCKER_TAG} develop" )
            pipeline.deploy("docker", "staging", "kubermatic", "deployment/kubermatic-api-v1", "api=kubermatic/api:${env.DOCKER_TAG}")
            pipeline.deploy("docker", "staging", "kubermatic", "deployment/cluster-controller-v1", "cluster-controller=kubermatic/api:${env.DOCKER_TAG}")
        } else {
            pipeline.dockerBuild("docker", "${env.DOCKER_TAG} dev" )
            pipeline.deploy("docker", "dev", "kubermatic", "deployment/kubermatic-api-v1", "api=kubermatic/api:${env.DOCKER_TAG}")
            pipeline.deploy("docker", "dev", "kubermatic", "deployment/cluster-controller-v1", "cluster-controller=kubermatic/api:${env.DOCKER_TAG}")
        }
}
