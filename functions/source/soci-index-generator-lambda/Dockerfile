# Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

FROM golang:1.19-alpine

RUN apk --no-cache add py3-pip make git zip gcc g++ zlib zlib-dev 

COPY . /build

WORKDIR /build

RUN GOPROXY=direct go mod download

RUN GOPROXY=direct make -f Makefile

CMD mkdir -p /output/ && mv /build/soci_index_generator_lambda.zip /output/
