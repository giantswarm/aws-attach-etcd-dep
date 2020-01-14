FROM alpine:3.9

RUN apk add --no-cache ca-certificates

ADD ./aws-attach-ebs-by-tag  /aws-attach-ebs-by-tag

ENTRYPOINT ["/aws-attach-ebs-by-tag"]
