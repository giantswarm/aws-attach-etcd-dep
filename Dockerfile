FROM alpine:3.11

RUN apk add --no-cache ca-certificates

ADD ./aws-attach-etcd-dep  /aws-attach-etcd-dep

ENTRYPOINT ["/aws-attach-etcd-dep"]
