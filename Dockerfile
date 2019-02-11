# Build image
FROM golang:1.11.5 AS build-env
WORKDIR /go/src/github.com/Jleagle/digital-ocean-ddns/
COPY . /go/src/github.com/Jleagle/digital-ocean-ddns/
RUN apk update && apk add curl git openssh
RUN curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh
RUN dep ensure
RUN CGO_ENABLED=0 GOOS=linux go build -a

# Runtime image
FROM alpine:3.9 AS runtime-env
WORKDIR /root/
COPY --from=build-env /go/src/github.com/Jleagle/digital-ocean-ddns/digital-ocean-ddns ./
COPY records.yaml ./records.yaml
RUN apk update && apk add ca-certificates curl bash
CMD ["./digital-ocean-ddns"]
