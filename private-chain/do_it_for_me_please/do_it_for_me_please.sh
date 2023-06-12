#!/bin/bash

clear

LIGHT_FOLDER=light_nodes
counter=0

rm -fr ./${LIGHT_FOLDER}/*
mkdir -p ${LIGHT_FOLDER}

# find the bridges and loop for each of them
for i in $(kubectl -n bridge get po --sort-by=.metadata.name -oname);do
    echo "=============================================================="
    echo $i
    # we are gonna use the counter to identify the LN sts
    ((counter++))
    kubectl -n bridge  logs $i |grep "The p2p host is listening on" -A2|grep "ip4"|tail -n+3|cut -d' ' -f3
    TRUSTED=$(kubectl -n bridge  logs $i |grep "The p2p host is listening on" -A4|grep "ip4"|grep "tcp"|cut -d' ' -f3)
    echo ${TRUSTED}
    TRUSTED=$(echo ${TRUSTED} | sed 's/\//\\\//g')

    # copy the base manifest
    cp sts_light.yaml sts_light_${counter}.yaml

    #sed -i '' "s/da-light-1/da-light-${counter}/g" sts_light_$counter.yaml
    # replacing the light-1 to the right number
    sed -i '' "s/light-1/light-${counter}/g" sts_light_$counter.yaml

    # replace whatever to the right value
    sed -i '' "s/WHATEVER/\"${TRUSTED}\"/g" sts_light_$counter.yaml

    # clean up double quotes added...
    sed -i '' 's/\[""/["/g' sts_light_$counter.yaml
    sed -i '' 's/\""]/"]/g' sts_light_$counter.yaml

    mv sts_light_${counter}.yaml ./${LIGHT_FOLDER}/
done
