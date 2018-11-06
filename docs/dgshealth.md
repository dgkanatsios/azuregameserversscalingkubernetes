# Dedicated Game Server Health [WIP]

There are cases in which your Dedicated Game Server (DGS) might be unhealthy. In these cases, it can report its DGSState via an API call. If the status is Failed, the DedicatedGameServerCollection controller will try and make an effort to recover the DGS by creating the new one in its place. The old Failed DGS can either be removed from the collection or deleted. The DGSCollection has two configurable fields about it:

```YAML
  dgsFailBehavior: Remove # or Delete
  dgsMaxFailures: 2
```

*dgsFailBehavior* dictates what will happen to a DGS when its DGSState is Failed. Possible values are 'Remove' and 'Delete'. Default value is 'Remove'.
*dgsMaxFailures* defines the maximum number of failures a DGSCollection can withstand. If another failure occurs, then the DGSCol enters the state of 'NeedsIntervention' in which basically the controller stops acting and a human intervention is required to examine and fix the collection.