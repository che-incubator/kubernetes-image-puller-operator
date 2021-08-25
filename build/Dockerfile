# Copyright (c) 2020-2021 Red Hat, Inc.
# This program and the accompanying materials are made
# available under the terms of the Eclipse Public License 2.0
# which is available at https://www.eclipse.org/legal/epl-2.0/
#
# SPDX-License-Identifier: EPL-2.0
#
# Contributors:
#   Red Hat, Inc. - initial API and implementation
#

# https://access.redhat.com/containers/?tab=tags#/registry.access.redhat.com/ubi8/go-toolset
FROM registry.access.redhat.com/ubi8/go-toolset:1.15.14-3 as builder
ENV GOPATH=/go/
USER root
ADD . /go/src/github.com/che-incubator/kubernetes-image-puller-operator
WORKDIR /go/src/github.com/che-incubator/kubernetes-image-puller-operator
RUN go test -v ./pkg... ./cmd... 
RUN go build -o build/_output/bin/kubernetes-image-puller-operator cmd/manager/main.go

# https://access.redhat.com/containers/?tab=tags#/registry.access.redhat.com/ubi8-minimal
FROM registry.access.redhat.com/ubi8-minimal:8.4-208

ENV OPERATOR=/usr/local/bin/kubernetes-image-puller-operator \
    USER_UID=1001 \
    USER_NAME=kubernetes-image-puller-operator

# install operator binary
COPY --from=builder /go/src/github.com/che-incubator/kubernetes-image-puller-operator/build/_output/bin/kubernetes-image-puller-operator ${OPERATOR}

COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}

# append Brew metadata here
