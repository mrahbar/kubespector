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
- You want to run network and scale tests on a new or already running cluster
- You want to perform maintenance task like create an `etcd` backup
- You want to check the status of you Kubernetes cluster
- You want to fetch logs from system services, Kubernetes pods or Docker Container alike
- You want to perform tasks from a Linux, Windows or Mac  

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

##  CLI command examples
1. Check the status of kubelet service on a group of nodes
    - ``./kubespector service status -g worker -s kubelet``
2. Delete unused Docker images on all worker nodes
    - ``./kubespector exec -g worker -c 'sudo docker images | grep demo-container | awk "{print $3}" | xargs --no-run-if-empty sudo docker rmi'``
3. Create an etcd backup with client-certificate
   - ``./kubespector etcd backup --secure --data-dir /opt/etcd --ca-cert /etc/etcd/certs/ca.crt --client-cert /etc/etcd/certs/server.crt --client-cert-key /etc/etcd/certs/server.key --endpoint https://128.0.64.211:2379 -o ./backup``  
4. Check the status of you Kubernetes cluster
    - ``./kubespector cluster-status``
5. Run network and scale tests
    - ``./kubespector performance network-test``    
    - ``./kubespector performance scale-test``    
6. Fetch logs from Docker daemon 
    - ``./kubespector logs -n kubernetesnode2 --element docker --type service --tail 5 -s -o ./docker.log``

## The Kubespector config file
Kubspector needs a config file generally named `kubespector.yml` which contains the ssh configuration as well as metadata about the cluster groups.
There are two predefined cluster groups: `Etcd` & `Master`. On top of that you can add your own cluster group like `Worker`, `Loadbalancer`, ...
There is also a special configuration called Kubernetes which will be explained later.
 
### SSH configuration
In this section the settings for a ssh connection can be configured. A minimal sample configuration looks like this:
````
Ssh:
  Connection:
    Username: <username>
    PrivateKey: <path to privat key>
    Port: 22
    Timeout: 90s
````
Following options can also be used inside _Connection:_
- AgentAuth: **true** or **false**
- HandshakeAttempts: integer number, default **3**
- FileTransferMethod: either **scp** or **sftp**

Additionally the ssh configuration supports local and bastion connection.
On a local connection kubespector assumes it will be on a node which is defined in a cluster group. Example:
 ````
 Ssh:
   Connection:
     Username: <username>
     PrivateKey: <path to privat key>
     Port: 22
     Timeout: 90s
   LocalOn:   
     Host: "kube-node1"
     IP: x.x.x.x 
 ````
 
For a bastion connection, kubespector tries to tunnel the ssh connections through the bastion server to the cluster.
The options for the ssh connect to the bastion server are the same as above. A sample configuration looks like this:
 ````
 Ssh:
   Connection:
     Username: <username>
     PrivateKey: <path to privat key>
     Port: 22
     Timeout: 90s
   Bastion:   
     Connection:
       Username: <username>
       PrivateKey: <path to privat key>
       Port: 22
       Timeout: 90s
     Node:
       Host: "kube-node1"
       IP: x.x.x.x 
 ````
### Cluster group configuration
````
  - Name: Etcd
    Services:
    - etcd
    Nodes:
    - Host: "kubernetesnode1"
      IP: 128.0.64.211
    - Host: "kubernetesnode2"
      IP: 128.0.64.212
    - Host: "kubernetesnode3"
      IP: 128.0.64.213
    Certificates:
    - /etc/kubernetes/certs/etcd/ca.crt
    - /etc/kubernetes/certs/etcd/client.crt
    DiskUsage:
      FileSystemUsage:
      - /dev/sda1
      DirectoryUsage:
      - /etc/etcd
      - /var/log
````

### Kubernetes section
````
Kubernetes:
  Resources:
  - Type: nodes
  - Type: pods
    Namespace: kube-system
    Wide: true
````

## Performance tests

## Acknowledgement
- kismatic (kubernetes-inspector idea)
- packer (communicator.go)