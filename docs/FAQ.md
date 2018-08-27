# Frequently asked questions

## Any recommendations about the "Nodes should have a Public IP" requirement?
Yup, check out [this](https://github.com/dgkanatsios/AksNodePublicIPController) project, it will probably help

## Inspiration about this project?
Check out a [project](https://github.com/dgkanatsios/AzureContainerInstancesManagement) I worked on some time ago. This uses Azure Container Instances to scale dedicated game servers on the Azure Cloud. Making a similar mechanism work with Kubernetes was the next logical step.

## How are game servers exposed to the Internet? 

Game Servers are loaded on each Node on a specific port (conceptually similar to docker run dedicatedgameserver -p X:Y). Port mapping is managed by our project.

## How did you end up using this networking solution? I know that Kubernetes has a thing called 'services' that allows exposing applications on the Internet (and a lot more).

Correct, [Kubernetes Services](https://kubernetes.io/docs/concepts/services-networking/service/) are a way to expose a set of Pods via a DNS name. Traffic sent to this Service is distributed to this set of Pods via a specified Load Balancing Algorithm. A certain type of Service, called [Load Balancer](https://kubernetes.io/docs/concepts/services-networking/service/#loadbalancer) allows exposing a set of Pods over the Internet, via a cloud provider's Load Balancer Service. In the case of Azure, a service called [Azure Load Balancer](https://docs.microsoft.com/en-us/azure/load-balancer/load-balancer-overview) is used.

In our case, each Dedicated Game Server is a single entity. There is no need for an extra layer to do the Load Balancing, since players are connecting directly to the game server. Moreover, the use of a Load Balancer Service was rejected because a) it would be an overkill to have a unique Load Balancer for each Dedicated Game Server and b) (most importantly) the presense of a Load Balancer would add unnecessary network hops, thus increasing the Latency. Another solution we tested was this of a [NodePort Service](https://kubernetes.io/docs/concepts/services-networking/service/#nodeport). This was abandoned as well because of the overhead of managing the Services. Consequently, the only remaining solution was to expose the Nodes to the Internet via Public IPs. AKS does not allow that by default in the time of writing, so we wrote [this](https://github.com/dgkanatsios/AksNodePublicIPController) utility to implement this functionality. 

First effort was to use `hostPort` functionality for each Pod, so we would hook up the container to each Node's network layer. This would require us to have the Dedicated Game Server listen to a specific port, which could easily become an issue in the future. So, we ended up using Kubernetes `hostPort` for each Pod. This way, we manage the Port mapping on each Node ourselves for each Pod.