#!/bin/bash


LIGHT_FOLDER=light_nodes
counter=0

rm -fr ./${LIGHT_FOLDER}/*
mkdir -p ${LIGHT_FOLDER}


for i in $(kubectl -n bridge get po --sort-by=.metadata.name -oname);do
    echo "=========================="
    echo $i
    ((counter++))
    kubectl -n bridge  logs $i |grep "The p2p host is listening on" -A2|grep "ip4"|tail -n+3|cut -d' ' -f3
   # TRUSTED=$(kubectl -n bridge  logs $i |grep "The p2p host is listening on" -A4|grep "ip4"|grep "tcp"|cut -d' ' -f3)
    TRUSTED=$(kubectl -n bridge  logs $i |grep "The p2p host is listening on" -A4|grep "ip4"|grep "tcp"|cut -d' ' -f3)
    echo ${TRUSTED}
    TRUSTED=$(echo ${TRUSTED} | sed 's/\//\\\//g')

    cp sts_light.yaml sts_light_${counter}.yaml

    sed -i '' "s/da-light-1/da-light-${counter}/g" sts_light_$counter.yaml
    sed -i '' "s/WHATEVER/\"${TRUSTED}\"/g" sts_light_$counter.yaml
    sed -i '' 's/\[""/["/g' sts_light_$counter.yaml
    sed -i '' 's/\""]/"]/g' sts_light_$counter.yaml

    mv sts_light_${counter}.yaml ./${LIGHT_FOLDER}/
done
