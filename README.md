# Walkthrough Operator

A Kubernetes Operator for creating services required by a given walkthrough.

# Deploying to a Cluster

Create namespace, rbac and CRDs
```
make install
```

Run the operator against a local openshift
```
make run
```

Create walkthough example cr
```
make create-examples
```

# Tear it down

```
make uninstall
```
