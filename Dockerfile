# This image is a tool for testing Docker images.
#
# The standard name for this image is openshift/imagetest
#
FROM openshift/origin-base

RUN yum install -y golang docker && yum clean all

RUN curl -L https://github.com/openshift/source-to-image/releases/download/v1.0/source-to-image-v1.0-77e3b72-linux-amd64.tar.gz | tar -xzC /usr/local/bin

ENV GOPATH /go

WORKDIR /go/src/

COPY imagetest.go /go/src/github.com/openshift/imagetest/

# usage: docker run --privileged -v /var/run/docker.sock:/var/run/docker.sock -v /path/to/test/dir:/go/src/test:ro openshift/imagetest -parallel=5 -v
# usage: docker run --privileged -v /var/run/docker.sock:/var/run/docker.sock -v `pwd`/2.0/test:/go/src/test:ro openshift/imagetest -parallel=5 -v
ENTRYPOINT ["go", "test", "test"]
