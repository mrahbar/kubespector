[![pipeline status](https://gitlab.com/mrahbar/kubernetes-inspector/badges/master/pipeline.svg)](https://gitlab.com/mrahbar/kubernetes-inspector/commits/master)
[![coverage report](https://gitlab.com/mrahbar/kubernetes-inspector/badges/master/coverage.svg)](https://gitlab.com/mrahbar/kubernetes-inspector/commits/master)
# kubespector - A cli tool to inspect your kubernetes cluster from remote
Kubespector was written out of the need to inspect a running kubernetes cluster remotely. 
It is packed with a lot of useful commands which make the live of Kubernetes admins easier.

Kubespector is opinionated about the type of Kubernetes installation (see Config section) but in the most parts it is configurable to your needs.
Since the remote connection uses ssh, it should be checked beforehand that the host running Kubespector can ssh into the target hosts.

#### Who should use kubespector?
- You frequently interact with system services e.g. check the status, restart a service
- You want to perform the same command on a set of nodes 
- You want to fetch logs from system services, Kubernetes pods or Docker Container alike
- You want to run network and scale tests on a new or already running cluster

## Usage
Call `kubespector` without any arguments to get an overview on commands. Some usage examples:


````
Kubespector can perform various actions on a Kubernetes cluster via ssh.

Usage:
  kubespector [command]

Available Commands:
  cluster-status Performs various checks on the cluster defined in the configuration file
  etcd           Executes various actions on a etcd cluster
  exec           Executes a command on a target group or node
  help           Help about any command
  kubectl        Wrapper for kubectl
  logs           Retrieve logs
  performance    Executes various performance tests
  scp            Secure bidirectional file copy
  service        Execute various actions on system services
  version        Prints the current version and build date

Flags:
  -f, --config string      Path to config file (default "./kubespector.yaml")
  -d, --debug              Set log-level to DEBUG
  -h, --help               help for kubespector
      --log-level string   Logging level, valid values: CRITICAL,ERROR,WARNING,INFO,DEBUG,TRACE (default "INFO")

Use "kubespector [command] --help" for more information about a command.
````

##  CLI commands

## Acknowledgement
- kismatic (kubernetes-inspector idea)
- packer (communicator.go)