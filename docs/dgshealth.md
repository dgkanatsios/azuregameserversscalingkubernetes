# Dedicated Game Server Health

There are cases in which your Dedicated Game Server (DGS) might be unhealthy. In these cases, it can report its *DGSHealth* via the `setsdgshealth` API call. If the health state is Failed, the DedicatedGameServerCollection controller will try and make an effort to recover the DGS by creating a new one in its place. The old (Failed) DGS can either be removed from the collection or be deleted. The DGSCollection has two configurable fields about this behavior:

```YAML
  dgsFailBehavior: Remove # or Delete
  dgsMaxFailures: 2
```

*dgsFailBehavior* dictates what will happen to a DGS when its DGSHealth is Failed. Possible values are 'Remove' and 'Delete', with 'Remove' being the default one.
*dgsMaxFailures* defines the maximum number of failures a DGSCollection can withstand. If the total number of failures is equal to dgsMaxFailures and another DGS becomes Failed, then the DGSCol will be assigned a health state called 'NeedsIntervention'. Here, the DGSCol controller stops working and a human intervention is required to examine and repair the collection and the DGSs in it.