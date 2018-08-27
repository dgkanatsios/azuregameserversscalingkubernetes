[![unofficial Google Analytics for GitHub](https://gaforgithub.azurewebsites.net/api?repo=docker_openarena_k8s)](https://github.com/dgkanatsios/gaforgithub)
# docker_openarena_k8s
OpenArena server - docker image for Kubernetes

This is a docker image with an OpenArena server. This image utilizing OpenArena 0.8.8's features.
A fork of https://github.com/sago007/docker_openarena that works with [AzureGameServersScalingKubernetes](https://github.com/dgkanatsios/AzureGameServersScalingKubernetes) project by extending it with:

- ability to set server name via env variable ($SERVER_NAME)
- stores connected users count to /tmp/connected

TODO:

- posts updates about connected users count to Azure Function (url is $SET_SESSIONS_URL, set during container creation from the AzureContainerInstancesManagement project).

To run locally, type:

```bash
docker build -t dgkanatsios/docker_openarena_k8s .
#docker run --rm -it -p 27960:27960/udp -e OA_STARTMAP=dm4ish -e OA_PORT=27960 -e SET_SESSIONS_URL=https://teeworlds.azurewebsites.net/api/ACISetSessions?code=<KEY> -e RESOURCE_GROUP='openarena' -e CONTAINER_GROUP_NAME='openarenaserver1' --name openarenaserver1 -v PATH/TO/openarena_data:/data dgkanatsios/docker_openarena
```

Docker Hub link: https://hub.docker.com/r/dgkanatsios/docker_openarena_k8s/

# Environment variables
There are 3 variables that can be set:

 * OA_STARTMAP - The the first map that the server loads (default dm4ish)
 * OA_PORT - The port that the game listens on 
 * OA_ROTATE_LOGS - Should the log be rotated? (default 1 = true)
 * SERVER_NAME

# Log rotation
If the environment OA_ROTATE_LOGS is set to "1" (witch is the default value) then "games.log" will be rotated up to once a day if the size exceeds ~50 MB. The logs will only be rotated on startup/restart. Old logs will be stored in the format "games.log.YYYY-MM-DD.gz" (this is the reason that we can only store once a day).
