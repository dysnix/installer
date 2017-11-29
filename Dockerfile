FROM golang:1.8

WORKDIR /go/src/git.arilot.com/kuberstack/kuberstack-installer

RUN mkdir -p /var/lib/kuberstack-installer/tmp

ADD . /go/src/git.arilot.com/kuberstack/kuberstack-installer

# Install requirements
RUN go-wrapper download github.com/jteeuwen/go-bindata/...
RUN go-wrapper download github.com/go-swagger/go-swagger/cmd/swagger
RUN go-wrapper install github.com/jteeuwen/go-bindata/...
RUN go-wrapper install github.com/go-swagger/go-swagger/cmd/swagger

# Go generation
RUN cd /go/src/git.arilot.com/kuberstack/kuberstack-installer/protocol \
    && mkdir gen && go generate && git apply configureAPI.patch

RUN cd /go/src/git.arilot.com/kuberstack/kuberstack-installer/predefined \
    && mkdir gen && go generate

# Build and install
RUN go-wrapper download git.arilot.com/kuberstack/kuberstack-installer/protocol/gen/cmd/kuberstack-installer-server
RUN go-wrapper install git.arilot.com/kuberstack/kuberstack-installer/protocol/gen/cmd/kuberstack-installer-server


# Run tests
RUN go test $(go list ./... | grep -v /vendor/ | grep -v /gen/ )

EXPOSE 8080
VOLUME /var/lib/kuberstack-installer/

CMD /go/bin/kuberstack-installer-server \
    --scheme=http \
    --host=0.0.0.0 \
    --port=8080 \
    --tmpDir=/var/lib/kuberstack-installer/tmp \
    --dbURI=/var/lib/kuberstack-installer/kuberstack-installer.db
