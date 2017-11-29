# Kuberstack Installer

## What is Kuberstack Installer?

Kuberstack-installer is a web-interface for Kubernetes cluster install and management.

## Features:

* Install Kubernetes production-ready cluster based on [KOPS](https://github.com/kubernetes/kops)
* Deploy into AWS
* Customization Kubernetes Cluster installation
* High Availability, Multizone Configuration

## Run Installer backend

    docker run -p 127.0.0.1:8080:8080 kuberstack/installer

API is available on URL http://localhost:8080

## TODO
* Easy management of Kubernetes cluster
* Management of bunch of cluster
* Instance Groups management
* One-click install additional software (Kubernetes Dashboard, Heapster, Autoscaler, Helm, Gitlab, etc.)
* CI/СD integration
* Security (Bastion host)
* Easy Kuberntes upgrade
* Cluster Federation

## Links
* https://kuberstack.com
