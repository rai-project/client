FROM webgpu/cuda:ppc64le-8.0-devel


ENV CUDNN_VERSION 6
LABEL com.nvidia.cudnn.version="6"

ADD http://developer.download.nvidia.com/compute/redist/cudnn/v6.0/cudnn-8.0-linux-ppc64le-v6.0.tgz /usr/local
RUN ldconfig
