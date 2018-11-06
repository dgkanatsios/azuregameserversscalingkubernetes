# Autoscaling

## DgsAutoscaler

Project contains an **experimental** Dedicated Game Server autoscaler controller. This autoscaler can scale DedicatedGameServer instances within a DedicatedGameServerCollection. Its usage is optional and can be configured during the deployment of a DedicatedGameServerCollection resource. The autoscaler lives on the aks-gaming-controller executable and can be optionally enabled.
The decision about whether there should be a scaling activity is determined based on the `ActivePlayers` metric. We take into account that each DedicatedGameServer can hold a specific amount of players. If the sum of the active players on all the running servers of the DedicatedGameServerCollection is above a specified threshold (or below, for scale in activity), then the system is clearly in need of more DedicatedGameServer instances, so a scale out activity will occur, increasing the requested replicas of the DedicatedGameServerCollection by one. Moreover, there is a cooldown timeout so that a minimum amount of time will pass between two successive scaling activities.
Here you can see a configuration example, fields are self-explainable:

```yaml
# field of DedicatedGameServerCollection.Spec
dgsAutoScalerDetails:
  minimumReplicas: 5
  maximumReplicas: 10
  scaleInThreshold: 60
  scaleOutThreshold: 80
  enabled: true
  coolDownInMinutes: 5
  maxPlayersPerServer: 10
```