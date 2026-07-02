// Declarative Jenkins pipeline mirroring the GitHub Actions workflow.
//
// Requirements on the Jenkins agent:
//   - Docker (with buildx) available to the agent
//   - Go 1.23 toolchain (or run the test stage in the golang:1.23 image)
//   - kubectl + kustomize for the deploy stage
//   - Credentials configured in Jenkins:
//       * 'registry-credentials' (username/password) for the image registry
//       * 'kubeconfig' (secret file) for cluster access
pipeline {
  agent any

  options {
    timestamps()
    timeout(time: 30, unit: 'MINUTES')
    disableConcurrentBuilds()
    buildDiscarder(logRotator(numToKeepStr: '20'))
  }

  environment {
    REGISTRY   = 'ghcr.io'
    IMAGE_NAME = 'OWNER/devops-showcase'
    // Short git SHA for immutable image tags.
    GIT_SHA    = "${env.GIT_COMMIT ? env.GIT_COMMIT.take(7) : 'local'}"
    IMAGE      = "${REGISTRY}/${IMAGE_NAME}"
  }

  stages {
    stage('Checkout') {
      steps {
        checkout scm
      }
    }

    stage('Test & Lint') {
      agent {
        docker {
          image 'golang:1.23'
          reuseNode true
          args '-v $HOME/.cache:/root/.cache'
        }
      }
      steps {
        sh 'go vet ./...'
        sh 'go test -race -covermode=atomic -coverprofile=coverage.out ./...'
      }
      post {
        always {
          archiveArtifacts artifacts: 'coverage.out', allowEmptyArchive: true
        }
      }
    }

    stage('Build image') {
      steps {
        sh """
          docker build \
            --build-arg VERSION=${GIT_SHA} \
            --build-arg COMMIT=${GIT_SHA} \
            --build-arg BUILD_DATE=\$(date -u +%Y-%m-%dT%H:%M:%SZ) \
            -t ${IMAGE}:${GIT_SHA} \
            -t ${IMAGE}:latest \
            .
        """
      }
    }

    stage('Scan image (Trivy)') {
      steps {
        // Report vulnerabilities; does not fail the build by default.
        sh """
          docker run --rm \
            -v /var/run/docker.sock:/var/run/docker.sock \
            aquasec/trivy:latest image \
            --severity HIGH,CRITICAL --exit-code 0 \
            ${IMAGE}:${GIT_SHA}
        """
      }
    }

    stage('Push image') {
      when { branch 'main' }
      steps {
        withCredentials([usernamePassword(
          credentialsId: 'registry-credentials',
          usernameVariable: 'REG_USER',
          passwordVariable: 'REG_PASS')]) {
          sh """
            echo "\$REG_PASS" | docker login ${REGISTRY} -u "\$REG_USER" --password-stdin
            docker push ${IMAGE}:${GIT_SHA}
            docker push ${IMAGE}:latest
          """
        }
      }
    }

    stage('Validate manifests') {
      steps {
        sh 'kubectl kustomize k8s/overlays/staging   > staging.rendered.yaml'
        sh 'kubectl kustomize k8s/overlays/production > production.rendered.yaml'
        archiveArtifacts artifacts: '*.rendered.yaml', allowEmptyArchive: true
      }
    }

    stage('Deploy to production') {
      when { branch 'main' }
      steps {
        withCredentials([file(credentialsId: 'kubeconfig', variable: 'KUBECONFIG')]) {
          dir('k8s/overlays/production') {
            sh "kustomize edit set image ghcr.io/OWNER/devops-showcase=${IMAGE}:${GIT_SHA}"
          }
          sh 'kubectl apply -k k8s/overlays/production'
          sh 'kubectl -n devops-showcase rollout status deployment/devops-showcase --timeout=120s'
        }
      }
    }
  }

  post {
    success { echo "Pipeline succeeded for ${IMAGE}:${GIT_SHA}" }
    failure { echo "Pipeline failed. Check the stage logs above." }
    always  { cleanWs() }
  }
}
