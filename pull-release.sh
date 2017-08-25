#!/bin/bash
private_token=zUa6dSCDumYiSMAUwzyx
job_id=$(curl --silent -g -L "https://gitlab.com/api/v4/projects/3838156/jobs?private_token=$private_token&scope[]=success&per_page=1&page=1" | python -c "import sys, json; print json.load(sys.stdin)[0]['id']")

if [ $? -eq 0 ]; then
    echo "Downloading release of job with id $job_id"
    curl -o kubespector.zip -L "https://gitlab.com/mrahbar/kubernetes-inspector/-/jobs/$job_id/artifacts/download?private_token=$private_token"
    mkdir release-$job_id
    unzip -B kubespector.zip -d release-$job_id/
    rm kubespector.zip
else
    echo "Could not retrieve last successfull job id
fi