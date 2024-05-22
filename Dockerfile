FROM alpine:3.20

RUN apk add --no-cache ca-certificates e2fsprogs util-linux

ADD ./aws-attach-etcd-dep  /aws-attach-etcd-dep

ENTRYPOINT ["/aws-attach-etcd-dep"]
