# Architecture

## Kubernetes Custom Resource Definitions

First of all, if you don't know what Kubernetes is and what it can do, we'd encourage you to check [here](https://kubernetes.io/docs/concepts/overview/what-is-kubernetes/) for the official documentation or watch [this](https://www.youtube.com/watch?v=4ht22ReBjno) great introductory video.

We are extending Kubernetes by using something called [Custom Resource Definition (CRDs)](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/) objects. These objects will be used to represent our dedicated game server entities. 

Specifically, we have two core entities in our project, which are represented by two respective CRDs:

- **DedicatedGameServer** ([YAML](/artifacts/crds/dedicatedgameserver.yaml), [Go](/pkg/apis/azuregaming/v1alpha1/dedicatedgameserver.go)): this represents the multiplayer game server itself. Each DedicatedGameServer has a single corresponding child [Pod](https://kubernetes.io/docs/concepts/workloads/pods/pod/) which will run the container image with your game server executable.
- **DedicatedGameServerCollection** ([YAML](/artifacts/crds/dedicatedgameservercollection.yaml), [Go](/pkg/apis/azuregaming/v1alpha1/dedicatedgameservercollection.go)): this represents a collection/set of related DedicatedGameServers that will run the same Pod template and can be scaled in/out within the collection (i.e. add or remove more instances of them). Dedicated Game Servers that are members of the same Collection have a lot of similarities in their execution environment, e.g. all of them could launch the same multiplayer map or the same type of game. So, you could have one collection for a "Capture the flag" mode of your game and another collection for a "Conquest" mode. Or, a collection for players playing on map "X" and a collection for players playin on map "Y".

When you create a new DedicatedGameServerCollection definition file, these are the fields you need to declare:

- **replicas** (integer): number of requested DedicatedGameServer instances
- **portsToExpose** (array of integers): these are the ports that you want to be exposed in the [Worker Node/VM](https://kubernetes.io/docs/concepts/architecture/nodes/) when the Pod is created. The way this works is that each Pod you create will have >=1 number of containers. There, each container will have its own *Ports* definition. If a port in this definition is included in the *portsToExpose* array, this port will be publicly exposed in the Node/VM. This is accomplished by the creation of a **hostPort** value on the Pod's definition. The ports' management is a procedure that is managed exclusively by our solution
- **template** (PodSpec): this is the actual Kubernetes [Pod template](https://kubernetes.io/docs/concepts/workloads/pods/pod-overview/#pod-templates) that holds information about the Pod's containers, ports, images etc.

For example YAML files, feel free to take a look in the `artifacts/examples` folder.

### Solution Components

This solution contains 2 main components, both of which are created as a single instance Kubernetes [Deployments](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/) in the namespace **dgs-system**:

#### API Server components

This is our project's API server *(that has nothing to do with Kubernetes API Server)*. It contains two sub-components, our project's own API Server as well as a Kubernetes [admission webhook](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#admission-webhooks) to validate and mutate incoming DGS requests.

##### API Server subcomponent

The API server subcomponent provides REST APIs that can be called by either the game server or an external scheduling system (e.g. a lobby service or a matchmaker). These APIs are split into two categories:

The first category contains HTTP methods that are to be called by the Dedicated Game Servers:

- **/setactiveplayers**: This method allows the dedicated game server to notify the API Server about currently connected players.
Definition of the POST data is:
```go
type ServerActivePlayers struct {
	ServerName  string `json:"serverName"`
	Namespace   string `json:"namespace"`
	PlayerCount int    `json:"playerCount"`
}
```

- **/setsdgshealth**: This method allows the dedicated game server to notify the API Server about the health of the respective DGS.
Definition of the POST data is:
```go
type ServerHealth struct {
	ServerName string `json:"serverName"`
	Namespace  string `json:"namespace"`
	Health     string `json:"health"`
}
```

`health` field can have one of these four values:
- Creating
- Healthy
- Failed

- **/setdgsmarkedfordeletion**: This method allows the dedicated game server to notify the API Server when the DGS is MarkedForDeletion (i.e. it will be deleted when there are 0 active players playing the game).
Definition of the POST data is:
```go
type ServerMarkedForDeletion struct {
	ServerName        string `json:"serverName"`
	Namespace         string `json:"namespace"`
	MarkedForDeletion bool   `json:"markedForDeletion"`
}
```

- **/setdgsstate**: This methods allows the DedicatedGameServer to notify the API Server about the state of the game itself.
Definition of the POST data is:
```go
type ServerState struct {
	ServerName string `json:"serverName"`
	Namespace  string `json:"namespace"`
	State      string `json:"state"`
}
```

`state` field can have one of these values:
- Idle *DGS has been created and not assigned yet to a match*
- Assigned *a match has been assigned to the DGS. DGS is currently waiting for players*
- Running *game is running*
- PostMatch *game has finished*

Bear in minnd that it is strictly the responsibility of either the DGS or of the external service (e.g. matchmaker/lobby) to modify the DGS state using one of the mentioned values.

The second category contains these HTTP methods:

- **/create**: This will create a new DedicatedGameServerCollection instance
- **/delete**: This will delete a DedicatedGameServerCollection instance
- **/running**: This will return all the available and running DedicatedGameServer instances in JSON format (i.e. it will return those DGSs that have the Pod "Running", the Health "Healthy" and are not MarkedForDeletion)

If the API Server is called on root URL (**/**) it will return an HTML page that displays data from the `/running` endpoint, so it can easily be accessed by a web browser.

All API methods are protected via an access code, represented as string and kept in a [Kubernetes Secret](https://kubernetes.io/docs/concepts/configuration/secret/) called `apiaccesscode`. This is created during project's installation and should be passed in all method calls `code` GET parameter. The only method that does not require authentication by default is the `/running` one. This, however, can be changed in the API Server process command line arguments.

##### Webhook subcomponent

The webhook component contains a Kubernetes [mutating admission webhook](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#admission-webhooks) which validates and modifies requests about our CRDs to the Kubernetes API Server. Specifically, it acts both as validating and a mutating admission webhook by performing these two operations:

- It checks if the Pods specified in the DedicatedGameServerCollection template have a [Resources section with CPU/Memory requests and limits](https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/#resource-requests-and-limits-of-pod-and-container). If the containers in the Pod lack this information, the webhook will reject the submission
- It mutates the Pods so as to add [Pod Affinity](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#affinity-and-anti-affinity) information. This helps the Kubernetes scheduler group the DedicatedGameServer Pods in Nodes consecutively, instead of distributing them in the cluster (which is - more or less - the behavior of the default Kubernetes scheduler).

#### Controller(s)

This component contains custom [Kubernetes controllers](https://github.com/kubernetes/sample-controller) for our CRDs. These controllers will perform various activities on the system when DedicatedGameServerCollections or DedicatedGameServers are created or updated. Moreover, there is an additional controller that handles the autoscaling part on each DedicatedGameServerCollection. This controller, called DGSAutoScalerController is optionally started. For more details, check out the [controllers documentation](controllers.md).

## Dedicated Game Server Health

There are cases in which your Dedicated Game Server (DGS) might be unhealthy. In these cases, it can report its *DGSHealth* via the `setsdgshealth` API call. If the health state is Failed, the DedicatedGameServerCollection controller will try and make an effort to recover the DGS by creating a new one in its place. The old (Failed) DGS can either be removed from the collection or be deleted. The DGSCollection has two configurable fields about this behavior:

```YAML
  dgsFailBehavior: Remove # or Delete
  dgsMaxFailures: 2
```

*dgsFailBehavior* dictates what will happen to a DGS when its DGSHealth is Failed. Possible values are 'Remove' and 'Delete', with 'Remove' being the default one.
*dgsMaxFailures* defines the maximum number of failures a DGSCollection can withstand. If the total number of failures is equal to dgsMaxFailures and another DGS becomes Failed, then the DGSCol will be assigned a health state called 'NeedsIntervention'. Here, the DGSCol controller stops working and a human intervention is required to examine and repair the collection and the DGSs in it.