#! /bin/sh

RootPath=$(cd $(dirname $0)/..; pwd)
GOPATH=/go
GoAppName=github.com/chubaofs/chubaofs
GoAppSrcPath=$GOPATH/src/$GoAppName
ServerBuildDockerImage="chubaofs/cfs-build:1.0"
ClientBuildDockerImage="chubaofs/centos-ltp:1.0"

# test & build
docker run --rm -v ${RootPath}:/go/src/github.com/chubaofs/chubaofs $ClientBuildDockerImage /bin/bash -c "cd $GoAppSrcPath && make build_server"
docker run --rm -v ${RootPath}:/go/src/github.com/chubaofs/chubaofs $ClientBuildDockerImage /bin/bash -c "cd $GoAppSrcPath && make build_client"

# start server
#docker-compose -f ${RootPath}/docker/docker-compose.yml up -d servers

# start client
#docker-compose -f ${RootPath}/docker/docker-compose.yml run client
