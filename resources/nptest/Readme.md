### Testing locally

-----------------------
docker build -t endianogino/netperf:1.1 .

docker save endianogino/netperf:1.1 | ssh root@kube-master01 'docker load'
docker save endianogino/netperf:1.1 | ssh root@kube-master02 'docker load'
	 
core:
docker run -p 5202:5202 endianogino/netperf:1.1 --mode=orchestrator

kube-master01:
docker run -p 5201:5201/udp -e orchestratorPort=5202 -e workerName=netperf-w1 -e workerPodIP=192.168.15.10 -e orchestratorPodIP=192.168.15.9 endianogino/netperf:1.1 --mode=worker

kube-master02:
docker run -p 5201:5201/udp -e orchestratorPort=5202 -e workerName=netperf-w2 -e workerPodIP=192.168.15.11 -e orchestratorPodIP=192.168.15.9 endianogino/netperf:1.1 --mode=worker

core:
docker run -p 5201:5201/udp -e orchestratorPort=5202 -e workerName=netperf-w3 -e workerPodIP=192.168.15.9  -e orchestratorPodIP=192.168.15.9 endianogino/netperf:1.1 --mode=worker