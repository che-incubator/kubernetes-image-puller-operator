# Copyright (c) 2020 Red Hat, Inc.
# This program and the accompanying materials are made
# available under the terms of the Eclipse Public License 2.0
# which is available at https://www.eclipse.org/legal/epl-2.0/
#
# SPDX-License-Identifier: EPL-2.0
#
# Contributors:
#   Red Hat, Inc. - initial API and implementation
#

# NOTE: using registry.access.redhat.com/rhel8/go-toolset does not work (user is requested to use registry.redhat.io)
# NOTE: using registry.redhat.io/rhel8/go-toolset requires login, which complicates automation
# NOTE: since updateBaseImages.sh does not support other registries than RHCC, update to RHEL8
# https://catalog.redhat.com/software/containers/ubi8/go-toolset/5ce8713aac3db925c03774d1
FROM registry.access.redhat.com/ubi8/go-toolset:1.13.4-32 AS build
ENV PATH=/opt/rh/go-toolset-1.13/root/usr/bin:${PATH} \
    GOPATH=/go/

USER root
ADD . /go/src/github.com/che-incubator/kubernetes-image-puller-operator
WORKDIR /go/src/github.com/che-incubator/kubernetes-image-puller-operator
RUN go test -v ./pkg... ./cmd... 
RUN go build -o build/_output/bin/kubernetes-image-puller-operator cmd/manager/main.go

# https://access.redhat.com/containers/?tab=tags#/registry.access.redhat.com/ubi8-minimal
FROM registry.access.redhat.com/ubi8-minimal:8.2-345

ENV OPERATOR=/usr/local/bin/kubernetes-image-puller-operator \
    USER_UID=1001 \
    USER_NAME=kubernetes-image-puller-operator

# install operator binary
COPY --from=build /go/src/github.com/che-incubator/kubernetes-image-puller-operator/build/_output/bin/kubernetes-image-puller-operator ${OPERATOR}

COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}

# append Brew metadata here
