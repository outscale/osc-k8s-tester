# Copyright 2019 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# enpoint https://osu.eu-west-2.outscale.com

FROM golang:1.12.7-stretch
RUN apt-get -y update && \
    apt-get -y install gdb jq awscli groff vim && \
    echo "add-auto-load-safe-path /usr/local/go/src/runtime/runtime-gdb.py" >> /root/.gdbinit


WORKDIR /go/src/github.com/kubernetes-sigs/aws-ebs-csi-driver
COPY ./aws-ebs-csi-driver .

WORKDIR /go/src/github.com/aws/aws-k8s-tester
COPY ./aws-k8s-tester .

WORKDIR /go/src/github.com/kubernetes-sigs/aws-ebs-csi-driver
RUN make -j 4 && \
    cp /go/src/github.com/kubernetes-sigs/aws-ebs-csi-driver/bin/aws-ebs-csi-driver /bin/aws-ebs-csi-driver

WORKDIR /go/src/github.com/aws/aws-k8s-tester
RUN cd /go/src/github.com/aws/aws-k8s-tester && make -j 4 && \
    cp /go/src/github.com/aws/aws-k8s-tester/bin/aws-k8s-tester-*-linux-amd64 /bin/aws-k8s-tester

WORKDIR /go/src/github.com/kubernetes-sigs/aws-ebs-csi-driver

ENTRYPOINT ["/bin/aws-ebs-csi-driver"]


