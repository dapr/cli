REM This script has the sequence of the steps to run manually on a Windows machine
REM Due to limitations of virtual machines and nested virtualization, 
REM the script might be best to run on a physical machine or relatively powerful virtual machine in cloud,
REM minikube is pretty heavy on resources.
REM Another option is to use dedicated k8s cluster.

REM Navigate to any empty directory and run the following commands.
REM Preferably, to run the first cleanup commands to have a clean dapr and minikube environment.
REM At the end of the script, you might want to clean the directory manually and remove cloned "quickstarts" repo.
dapr uninstall --all -k
minikube delete
minikube stop
minikube start
dapr init -k --runtime-version 1.15.0-rc.10 --dev
git clone https://github.com/dapr/quickstarts.git
cd quickstarts/tutorials/hello-kubernetes
git checkout origin/release-1.15
git branch --show-current
dapr run -f -k .
