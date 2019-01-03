# Frequently asked questions

## Any recommendations about the "Nodes should have a Public IP" requirement?

Yup, check out [this](https://github.com/dgkanatsios/AksNodePublicIPController) project, it's recommended. An alternative project that does the same task is [here](https://github.com/dgkanatsios/AksNodePublicIP).

## Inspiration about this project?

Check out a [project](https://github.com/dgkanatsios/AzureContainerInstancesManagement) that I worked on some time ago. This uses [Azure Container Instances](https://azure.microsoft.com/en-us/services/container-instances/) and [Azure Functions](https://functions.azure.com) to scale dedicated game servers on  Azure. Making a similar mechanism with Kubernetes was the next logical step.

## How are game servers exposed to the Internet? 

DGSs are crated on each Node on a specific port (or set of ports, depending on the server requirements) (conceptually similar to the command `docker run dedicatedgameserver -p X:Y`). Port assignment and mapping is managed by our project.

## How did you end up using this networking solution? I know that Kubernetes has a thing called 'Service' that allows exposing applications on the Internet (and a lot more).

[Kubernetes Services](https://kubernetes.io/docs/concepts/services-networking/service/) is a way to expose a set of Pods via a DNS name (and more). Traffic sent to a Service is distributed to a corresponding set of Pods via a specified Load Balancing algorithm. A certain type of Service, called [Load Balancer](https://kubernetes.io/docs/concepts/services-networking/service/#loadbalancer) allows exposing a set of Pods over the Internet, via the cloud provider's Load Balancer Service. On Azure, a service called [Azure Load Balancer](https://docs.microsoft.com/en-us/azure/load-balancer/load-balancer-overview) is used for this purpose.

In our case, each DGS is a single entity. There is no need for an extra layer for the Load Balancing and network traffic management, since game clients are connecting directly to the DGS. Moreover, the use of a Load Balancer Service was also discouraged because a) it would be an overkill to have a unique Load Balancer for each Dedicated Game Server and b) (most importantly) the presense of a Load Balancer would potentially add unnecessary network hops, thus probably increasing the network latency. Another solution we tested was this of a [NodePort Service](https://kubernetes.io/docs/concepts/services-networking/service/#nodeport). This was abandoned as well because of the overhead of managing the Service entities.
Consequently, the solution was to expose the Nodes to the Internet via Public IPs. AKS does not allow that by default in the time of writing, so we created [this](https://github.com/dgkanatsios/AksNodePublicIPController) utility to implement this functionality. 

Moreover, on the port assignment, our first effort was to use `hostNetwork` functionality for each Pod [link](http://alesnosek.com/blog/2017/02/14/accessing-kubernetes-pods-from-outside-of-the-cluster/), so we would hook up the container to each Node's network layer (=> no software NAT for our containers). This would require us to have the DGS listen to a specific port (assigned by our project). As you can easily understand, this could disqualify DGSs that can only listen to hardcoded ports. So, we ended up using Kubernetes `hostPort` for each Pod. What we do is set a manual port (or more, depending on the DGS) for each Pod that is mapped to the game server's original listening port. Mapping is made possible via software NAT (container networking).

## How can I view the Kubernetes Master control plane logs on AKS?

Check [here](https://docs.microsoft.com/en-us/azure/aks/view-master-logs).

## How can I view the kubelet logs in a AKS Node?

Check [here](https://docs.microsoft.com/en-us/azure/aks/kubelet-logs).

## How did you mock time in your code for the autoscaler tests? [or, what is this 'clock' field in some objects]

We needed to mock `time` object for our tests, check [this](https://medium.com/agrea-technogies/mocking-time-with-go-a89e66553e79) blog post for instructions.

## How can I visualize my cluster objects/state?

Apart from the [Kubernetes dashboard](https://docs.microsoft.com/en-us/azure/aks/kubernetes-dashboard), you can also use [Weave Scope](https://www.weave.works/docs/scope/latest/installing/#k8s).

```bash
# install Weave Scope
kubectl apply -f "https://cloud.weave.works/k8s/scope.yaml?k8s-version=$(kubectl version | base64 | tr -d '\n')"
# port-forward the dashboard
kubectl port-forward -n weave "$(kubectl get -n weave pod --selector=weave-scope-component=app -o jsonpath='{.items..metadata.name}')" 4040
# open localhost:4040 on your browser
```

## I see that you have a self-signed certificate for authentication with WebhookServer. How can I generate my own?

Easy enough, use openssl ([source](https://stackoverflow.com/questions/10175812/how-to-create-a-self-signed-certificate-with-openssl))

```bash
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -nodes -subj '/CN=aks-gaming-webhookserver.default.svc' -days 365 
```

## How can I get my Kubernetes API Server CABundle value used for Validating and Mutating webhooks?

Run this command ([source](https://medium.com/ibm-cloud/diving-into-kubernetes-mutatingadmissionwebhook-6ef3c5695f74)):

```bash
kubectl get configmap -n kube-system extension-apiserver-authentication -o=jsonpath='{.data.client-ca-file}' | base64 | tr -d '\n'
```

## Any tool to "smoke test" my AKS installation and see if everything is working as supposed to?

Check [this](https://github.com/dsalamancaMS/K8sSmokeTest/blob/master/smoke.sh) bash script. [These](https://github.com/malachma/supp-tools/tree/master/k8s) script might help in troubleshooting as well.

## Can I change the namespace that the solution components are created? 

Yes, but unfortunately at the time of writing it is hardcoded into the application (check the constants.go file), so you would need to recompile and redeploy the solution.

## My containers take time to load/how can I make them smaller?

You could potentially move some of your static assets out of the container image and have it hosted elsewhere, e.g. on an [Azure File Storage](https://azure.microsoft.com/en-us/services/storage/files/) account. This will allow you to have a smaller image. Beware though that you should pay attention when you upgrade your image.

## Project installation creates an external Load Balancer that opens public access to the API Server. How could I make the Load Balancer internal?

Project (mainly for demonstration purposes) creates a LoadBalancer Kubernetes Service for the project's API Server. Even though its methods are protected by a code (the one that's stored in a Secret), it would be wise to hide it from the public internet. To accomplish this, you can use an internal Load Balancer using the instructions [here](https://docs.microsoft.com/en-us/azure/aks/internal-lb).

## Any recommendations for hosting my private game server images?

Check [Azure Container Registry](https://azure.microsoft.com/en-us/services/container-registry/)

## Any alternatives to this project? What other options do I have?

A lot!
- for a fully managed approach, you might want to check [PlayFab Multiplayer Servers](https://api.playfab.com/blog/introducing-playfab-multiplayer-servers)
- if you want to use Azure Container Instances service, check [this](https://github.com/dgkanatsios/AzureContainerInstancesManagement) project
- if you want to use Azure Batch service, check [this](https://github.com/PoisonousJohn/gameserver-autoscaler) project to get started
- Google and Ubisoft are working on project [Agones](https://github.com/GoogleCloudPlatform/agones) which runs [absolutely fine](https://github.com/GoogleCloudPlatform/agones/tree/master/install#setting-up-an-azure-kubernetes-service-aks-cluster) on AKS