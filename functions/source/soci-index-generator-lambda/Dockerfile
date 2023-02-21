FROM golang:1.19

RUN apt-get -y update
RUN apt-get -y install python3-pip make git zip gcc g++ zlib1g zlib1g-dev

COPY . /build

WORKDIR /build

RUN GOPROXY=direct go mod download

RUN GOPROXY=direct make -f Makefile

CMD mkdir -p /output/ && mv /build/soci_index_generator_lambda.zip /output/
