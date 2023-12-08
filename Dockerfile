FROM alpine:3.19

RUN apk add --no-cache ca-certificates e2fsprogs util-linux

ADD ./aws-attach-etcd-dep  /aws-attach-etcd-dep

ENTRYPOINT ["/aws-attach-etcd-dep"]
