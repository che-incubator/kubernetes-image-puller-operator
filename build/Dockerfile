FROM golang:1.13.8 AS build

ADD . /go/src/github.com/che-incubator/kubernetes-image-puller-operator
WORKDIR /go/src/github.com/che-incubator/kubernetes-image-puller-operator
RUN go test -v ./pkg... ./cmd... 
RUN go build -o build/_output/bin/kubernetes-image-puller-operator cmd/manager/main.go


FROM registry.access.redhat.com/ubi8/ubi-minimal:latest

ENV OPERATOR=/usr/local/bin/kubernetes-image-puller-operator \
    USER_UID=1001 \
    USER_NAME=kubernetes-image-puller-operator

# install operator binary
COPY --from=build /go/src/github.com/che-incubator/kubernetes-image-puller-operator/build/_output/bin/kubernetes-image-puller-operator ${OPERATOR}

COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
