pipeline {
  agent {
    dockerfile {
      filename '.devcontainer/Dockerfile.dev'
    }
  }
  environment {
    PROXY_TEST_USERNAME     = credentials('proxy-test-username')
    PROXY_TEST_PASSWORD = credentials('proxy-test-password')
  }

  options { 
    disableConcurrentBuilds() 
  }

  stages {
    stage('Unit Test') {
      steps {
        sh 'go test -coverpkg=./... -coverprofile=coverage.out ./... -timeout 100s -parallel 4'
      }
    }

    stage('Coverage') {
      steps {
        sh 'go tool cover -html=coverage.out -o coverage.html'
        archiveArtifacts '*.html'
        sh 'echo "Coverage Report: ${BUILD_URL}artifact/coverage.html"'
        sh '''t=$(go tool cover -func coverage.out | grep total | tail -1 | awk \'{print substr($3, 1, length($3)-1)}\')
if [ "${t%.*}" -lt 80 ]; then 
    echo "Coverage failed ${t}/80"
    exit 1
fi'''
      }
    }

    stage('Main Race Condition') {
      steps {
        lock('multi_branch_server') {
          sh 'go run --race main.go -t https://servdown.com/ -d 1 -n 1500'
          sh 'go run --race main.go -config config/config_testdata/race_configs/step_assertions_stdout.json'
          sh 'go run --race main.go -config config/config_testdata/race_configs/step_assertions_stdout_json.json'
          sh 'go run --race main.go -config config/config_testdata/race_configs/capture_envs.json'
          sh 'go run --race main.go -config config/config_testdata/race_configs/global_envs.json'
          sh 'go test -race -run ^TestDynamicVariableRace$ go.ddosify.com/ddosify/core/scenario/scripting/injection'
        }
      }
    }

  }
  post {
    unstable {
      slackSend(channel: '#jenkins', color: 'danger', message: "${currentBuild.currentResult}: ${currentBuild.fullDisplayName} - ${BUILD_URL}")
    }

    failure {
      slackSend(channel: '#jenkins', color: 'danger', message: "${currentBuild.currentResult}: ${currentBuild.fullDisplayName} - ${BUILD_URL}")
    }

  }
}
