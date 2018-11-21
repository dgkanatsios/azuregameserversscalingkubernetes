# Controllers

Kubernetes controllers are objects that are active reconciliation processes. In simple words, this means that a controller watches an object (or a set of objects) for its desired state as well as its actual state. It actively compares these states and it makes every effort to bring the actual state to look like the desired state.

This project contains a set of controllers, each one having assigned the task to reconcile a specific set of objects. All controllers are based on the official [Kubernetes sample controller](https://github.com/kubernetes/sample-controller). You can also find some good advice on writing controllers [here](https://github.com/kubernetes/community/blob/master/contributors/devel/controllers.md). Controllers in this project were made in order to reconcile our Custom Resource Definition (CRD) objects, i.e. the DedicatedGameServerCollections and the DedicatedGameServers.

## DedicatedGameServerCollectionController

The DedicatedGameServerCollection controller has the duty of handling the DedicatedGameServer objects of a DedicatedGameServerCollection. It may create new DedicatedGameServers, it may set their Status "MarkedForDeletion" field as true and it will update the DedicatedGameServerCollection status as well. It does that by watching the DedicatedGameServerCollection CRD objects in the system. It also watches the DedicatedGameServer CRD objects (that belong to a DedicatedGameServerCollection). When there is a change in either of these objects, the controller performs the following steps (either in a single loop or multiple ones):

- checks the DedicatedGameServerCollection object's requested Replicas. If it's less than the available, controller will proceed in creating more DedicatedGameServer objects. If it's more, then the controller will mark the required DedicatedGameServer objects as 'MarkedForDeletion'.
- updates the DedicatedGameServerCollection status with i) the number of available replicas ii) the DedicatedGameServers (that belong to the DedicatedGameServerCollection) overall status iii) the Pod (that belong to the DedicatedGameServers) overall status

## DedicatedGameServerController

The DedicatedGameServerCollection controller has the dury of handling the Pods of a DedicatedGameServer object. It may create new pods, it may delete a DedicatedGameServer if it has zero players and its "MarkedForDeletion" field is true and it will update the DedicatedGameServer state. Controller accomplishes these tasks by watching the DedicatedGameServer CRD objects in the system. It also watches the Pods in the system (that belong to a DedicatedGameServer). When there is a change in either of these objects, the controller performs the following steps (either in a single loop or multiple ones):

- checks if the DedicatedGameServer has the 'MarkedForDeletion' field set to true and if the number of active players on this server is zero. If this is the case, then the controller requests the deletion of this DedicatedGameServer instance. This will delete the corresponding pod as well via the Kubernetes garbage collection system
- checks if there is a pod for the changed DedicatedGameServer. If there is not, the controller will create one
- if a pod exists, the controller gets to update the corresponding DedicatedGameServer with i) Node's Public IP, ii) Node Name and iii) Pod state

## PodAutoScalerController

The PodAutoScalerController controller is optionally created (via a command line argument on the controller) and is responsible for Pod Autoscaling on every DedicatedGameServerCollection that opts into the pod autoscaling mechanism. The controller performs scaling by querying requesting DedicatedGameServerCollections for their child DedicatedGameServers and checking their total ActivePlayers metric. If its value is not between requested threshold, then the controller will either do scale in or scale out.

The PodAutoScalerController performs this tasks by watching the DedicatedGameServerCollection CRD objects in the system. It also watches the DedicatedGameServers in the system (that belong to a DedicatedGameServerCollection). When there is a change in either of these objects, the controller performs the following steps (either in a single loop or multiple ones):

- checks if the DedicatedGameServerCollection has autoscaling enabled
- if true, it checks if its Pod and DedicatedGameServer overall state is Running (if it's in another state like Failed or Creating or Pending the controller shouldn't scale)
- checks the last time a scale in/out operation took place, as there is a cooldown period in the autoscaler's settings
- checks if the current amount of DedicatedGameServers is below a requested maximum or above a requested minumum (depending on whether the controller checks for scale out or scale in)
- if all of the above are true, then the controller aggregates the **ActivePlayers** field on the DedicatedGameServers that belong to the DedicatedGameServerCollection in question. If the sum is below or above a requested threshold (again depending on scale in or scale out), then the controller submits a change in the **Replicas** field of the DedicatedGameServerCollection (either add one or remove one). This, in turn, will be handled by the DedicatedGameServerCollection controller which will create or mark as deletion a single DedicatedGameServer