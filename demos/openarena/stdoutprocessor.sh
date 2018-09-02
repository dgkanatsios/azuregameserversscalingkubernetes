#!/bin/bash
echo "Start processing"

while IFS= read -r line
do
    #echo line so that docker can gather its logs from stdout
    echo $line
 
    #this is the server initialization
    init=$(echo $line | grep 'Opening IP socket' | wc -l)
    if [ $init -eq 1 ]
    then
        echo "About to send data for server status: {\"serverName\":\"$SERVER_NAME\", \"status\":\"Running\"}"
        wget -O- --post-data="{\"serverName\":\"$SERVER_NAME\", \"status\":\"Running\"}" --header=Content-Type:application/json "$SET_SERVER_STATUS_URL"
    fi

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
        toAdd=-$(</tmp/connected) #reset all players count to zero
    fi

    if [ $x -eq 1 ] || [ $y -eq 1 ] || [ $z -eq 1 ]
    then
        #get current connected count from the file
        connected=$(</tmp/connected)
        #((..)) is the way for integer arithmetics on bash
        connected=$(($connected+$toAdd))
        echo $connected > /tmp/connected

        echo "About to send data for active players: {\"serverName\":\"$SERVER_NAME\", \"playerCount\":$connected}"
        wget -O- --post-data="{\"serverName\":\"$SERVER_NAME\", \"playerCount\":$connected}" --header=Content-Type:application/json "$SET_ACTIVE_PLAYERS_URL"

    fi 
done

echo "Finished processing"
