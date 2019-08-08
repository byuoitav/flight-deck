FROM byuoitav/amd64-alpine
MAINTAINER Ranny Dandall <danny_randall@byu.edu>

ARG NAME
ENV name=${NAME}

RUN sed -i -e 's/v3\.7/v3.8/g' /etc/apk/repositories
RUN apk update

COPY ${name}-bin ${name}-bin
COPY version.txt version.txt

# add any required files/folders here

ENTRYPOINT ./${name}-bin
