#!/bin/bash
echo "Start processing"

while IFS= read -r line
do
    #echo line so that docker can gather its logs from stdout
 
    echo $line
    
    #client connection
    x=$(echo $line | grep 'ClientBegin:' | wc -l)
    toAdd=0
    if [ $x -eq 1 ]
    then
       toAdd=1
    fi
    
    #client disconnection
    y=$(echo $line | grep 'ClientDisconnect:' | wc -l)
    if [ $y -eq 1 ]
    then
        toAdd=-1
    fi
    
    #this takes place when the server changes map
    z=$(echo $line | grep 'AAS shutdown.' | wc -l)
    if [ $z -eq 1 ]
    then
        toAdd=-$(</tmp/connected) #reset all players
    fi

    if [ $x -eq 1 ] || [ $y -eq 1 ] || [ $z -eq 1 ]
    then
        #get current connected count from the file
        connected=$(</tmp/connected)
        #((..)) is the way for integer arithmetics on bash
        connected=$(($connected+$toAdd))
        echo $connected > /tmp/connected

        #following are specified on Docker image creation
        #SET_SESSIONS_URL=https://acimanagement.azurewebsites.net/api/ACISetSessions?code=<KEY>
        #RESOURCE_GROUP='openarena'
        #CONTAINER_GROUP_NAME='openarenarver1'

        echo "[{\"name\":\"$SERVER_NAME\", \"activeSessions\":$connected}]"
        #wget -O- --post-data="[{\"resourceGroup\":\"$RESOURCE_GROUP\", \"containerGroupName\":\"$CONTAINER_GROUP_NAME\", \"activeSessions\":$connected}]" --header=Content-Type:application/json "$SET_SESSIONS_URL"
        wget -O- --post-data="[{\"name\":\"$SERVER_NAME\", \"activeSessions\":$connected}]" --header=Content-Type:application/json "$SET_SESSIONS_URL"

    fi 
done

echo "Finished processing"
