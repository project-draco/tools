#!/bin/bash
cat repositories.json | jq -r '.[] | .data.search.edges[].node | "docker run --rm -v $HOME/.ssh:/root/.ssh kenja converter.sh https://github.com/"+.owner.login+"/"+.name+".git git@github.com:project-draco-hr/"+.name+".git && \\"'
