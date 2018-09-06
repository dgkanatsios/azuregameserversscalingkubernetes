# Frequently asked questions

## Any recommendations about the "Nodes should have a Public IP" requirement?

Yup, check out [this](https://github.com/dgkanatsios/AksNodePublicIPController) project, it will probably help. An alternative project that does the same task is [here](https://github.com/dgkanatsios/AksNodePublicIP).

## Inspiration about this project?

Check out a [project](https://github.com/dgkanatsios/AzureContainerInstancesManagement) I worked on some time ago. This uses [Azure Container Instances](https://azure.microsoft.com/en-us/services/container-instances/) and [Azure Functions](https://functions.azure.com) to scale dedicated game servers on the Azure Cloud. Making a similar mechanism with Kubernetes was the next logical step.

## How are game servers exposed to the Internet? 

Game Servers are loaded on each Node on a specific port (conceptually similar to docker run dedicatedgameserver -p X:Y). Port mapping is managed by our project.

## Why are you keeping duplicate values (both in CRD .Spec and in Labels) about ActivePlayers, PodState and DedicatedGameServer?

We need to be able to query them via our APIServer. Currently, you can GET normal K8s objects via filters [source](https://kubernetes.io/docs/concepts/overview/working-with-objects/field-selectors/), however there is an [open issue](https://github.com/kubernetes/kubernetes/issues/53459) regarding CRDs.

## How did you end up using this networking solution? I know that Kubernetes has a thing called 'services' that allows exposing applications on the Internet (and a lot more).

Correct, [Kubernetes Services](https://kubernetes.io/docs/concepts/services-networking/service/) are a way to expose a set of Pods via a DNS name. Traffic sent to this Service is distributed to this set of Pods via a specified Load Balancing Algorithm. A certain type of Service, called [Load Balancer](https://kubernetes.io/docs/concepts/services-networking/service/#loadbalancer) allows exposing a set of Pods over the Internet, via a cloud provider's Load Balancer Service. In the case of Azure, a service called [Azure Load Balancer](https://docs.microsoft.com/en-us/azure/load-balancer/load-balancer-overview) is used.

In our case, each Dedicated Game Server is a single entity. There is no need for an extra layer to do the Load Balancing, since players are connecting directly to the game server. Moreover, the use of a Load Balancer Service was rejected because a) it would be an overkill to have a unique Load Balancer for each Dedicated Game Server and b) (most importantly) the presense of a Load Balancer would add unnecessary network hops, thus increasing the Latency. Another solution we tested was this of a [NodePort Service](https://kubernetes.io/docs/concepts/services-networking/service/#nodeport). This was abandoned as well because of the overhead of managing the Services. Consequently, the only remaining solution was to expose the Nodes to the Internet via Public IPs. AKS does not allow that by default in the time of writing, so we wrote [this](https://github.com/dgkanatsios/AksNodePublicIPController) utility to implement this functionality. 

First effort was to use `hostNetwork` functionality for each Pod [link](http://alesnosek.com/blog/2017/02/14/accessing-kubernetes-pods-from-outside-of-the-cluster/), so we would hook up the container to each Node's network layer (=> no software NAT for our containers). This would require us to have the Dedicated Game Server listen to a specific port, so each game server we would use would have to be activated in a different port for every running Pod. As you can understand, this could easily become an issue in the future since it might not be possible to customize listening port for a game server. So, we ended up using Kubernetes `hostPort` for each Pod. We set a manual port (or more) for each Pod that is mapped to the game server's original listening port. Mapping itself is made possible via software NAT. The tricky part is that we manage the Port mapping for each Pod on each Node ourselves.

## How can I view the Kubernetes Master control plane logs?

Check [here](https://docs.microsoft.com/en-us/azure/aks/view-master-logs).

## How can I view the kubelet logs in a AKS Node?

Check [here](https://docs.microsoft.com/en-us/azure/aks/kubelet-logs).

## How did you mock time in your code? [or, what is this 'clock' field in some objects]

We needed to mock `time` object for our tests, check [this](https://medium.com/agrea-technogies/mocking-time-with-go-a89e66553e79) blog post for instructions.
