#! /bin/sh
#ugly hack to force image re-pull
#https://github.com/kubernetes/kubernetes/issues/27081#issuecomment-238078103
kubectl patch deploy docker-openarena-k8s-api -p "{\"spec\":{\"template\":{\"metadata\":{\"labels\":{\"date\":\"`date +'%s'`\"}}}}}"
kubectl patch deploy docker-openarena-k8s-controller -p "{\"spec\":{\"template\":{\"metadata\":{\"labels\":{\"date\":\"`date +'%s'`\"}}}}}"