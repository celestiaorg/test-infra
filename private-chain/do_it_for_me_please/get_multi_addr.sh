#!/bin/bash

clear

LIGHT_FOLDER=light_nodes
counter=0
FILE=/tmp/trusted_peers.txt
FILE_SHORT=/tmp/trusted_peers_short.txt

if [ -f $FILE ];then
    rm $FILE
fi
if [ -f $FILE_SHORT ];then
    rm $FILE_SHORT
fi

# find the bridges and loop for each of them
for i in $(kubectl -n bridge get po --sort-by=.metadata.name -oname);do
    # we are gonna use the counter to identify the LN sts
    ((counter++))
    kubectl -n bridge  logs $i |grep "The p2p host is listening on" -A2|grep "ip4"|tail -n+3|cut -d' ' -f3
    TRUSTED=$(kubectl -n bridge  logs $i |grep "The p2p host is listening on" -A4|grep "ip4"|grep "tcp"|cut -d' ' -f3)
    TRUSTED_SHORT=$(echo $TRUSTED| cut -d'/' -f7)
    echo ${TRUSTED}
    echo ${TRUSTED_SHORT}

    echo ${TRUSTED} >> $FILE
    echo ${TRUSTED_SHORT} >> $FILE_SHORT
done
