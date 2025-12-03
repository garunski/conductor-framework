package testdata

import (
	"embed"
)

// TestManifests contains sample Kubernetes manifests for testing
var TestManifests = map[string]string{
	"service": `apiVersion: v1
kind: Service
metadata:
  name: test-service
  namespace: default
spec:
  ports:
  - port: 8080
  selector:
    app: test-app
`,
	"deployment": `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-deployment
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      app: test-app
  template:
    metadata:
      labels:
        app: test-app
    spec:
      containers:
      - name: test
        image: test:latest
        env:
        - name: ENV1
          value: value1
`,
	"statefulset": `apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: test-statefulset
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      app: test-app
  template:
    metadata:
      labels:
        app: test-app
    spec:
      containers:
      - name: test
        image: test:latest
`,
}

// EmptyFS is an empty embedded filesystem for testing
var EmptyFS embed.FS

// DefaultTestAppName is the default application name for tests
const DefaultTestAppName = "test-app"

// DefaultTestVersion is the default version for tests
const DefaultTestVersion = "test-version"

// DefaultTestNamespace is the default namespace for tests
const DefaultTestNamespace = "default"

