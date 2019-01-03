#!/bin/bash

kubectl create -f dedicatedgameservercollection.yaml

# demo scale out

# get DGS names
dgs=`kubectl get dgs -l DedicatedGameServerCollectionName=openarena | cut -d ' ' -f 1 | sed 1,1d`
# update DGS.Spec.ActivePlayers
kubectl patch dgs $dgs -p '[{ "op": "replace", "path": "/status/activePlayers", "value": 9 },]' --type='json'

# edit the last scaling operation datetime
kubectl edit dgsc openarena

# demo scale in
# get DGS names - again
dgs=`kubectl get dgs -l DedicatedGameServerCollectionName=openarena | cut -d ' ' -f 1 | sed 1,1d`
# update DGS.Spec.ActivePlayers
kubectl patch dgs $dgs -p '[{ "op": "replace", "path": "/status/activePlayers", "value": 5 },]' --type='json'
