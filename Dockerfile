FROM artifactory.dev.bi.zone:5000/sensors/bz_golang:1.14.1-lint1.24.0 AS builder

ARG URL
ARG LOGIN
ARG PASSWORD

COPY . /app
WORKDIR /app
RUN /app/build.sh \
    && curl -k --user "${LOGIN}:${PASSWORD}" ${URL} -o ./license.v2c

FROM alpine:latest
COPY --from=builder /app/packages/linux_amd64/uploadlicense /uploadlicense
COPY --from=builder /app/license.v2c /license.v2c

RUN mkdir /lib64 \
    && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2 \ 
    && chmod +x /uploadlicense \ 
    && chmod +x /license.v2c
ENTRYPOINT [ "/uploadlicense","--path=/license.v2c" ]
