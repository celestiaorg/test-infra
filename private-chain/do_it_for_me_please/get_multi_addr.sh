#!/bin/bash

clear

LIGHT_FOLDER=light_nodes
counter=0

# find the bridges and loop for each of them
for i in $(kubectl -n bridge get po --sort-by=.metadata.name -oname);do
    # we are gonna use the counter to identify the LN sts
    ((counter++))
    kubectl -n bridge  logs $i |grep "The p2p host is listening on" -A2|grep "ip4"|tail -n+3|cut -d' ' -f3
    TRUSTED=$(kubectl -n bridge  logs $i |grep "The p2p host is listening on" -A4|grep "ip4"|grep "tcp"|cut -d' ' -f3)
    echo ${TRUSTED}
done
