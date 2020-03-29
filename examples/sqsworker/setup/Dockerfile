FROM ubuntu:18.04

LABEL maintainer=dirk.pahl@web.de

RUN apt-get update
RUN apt-get install -y python-pip jq groff
RUN pip install awscli-local

COPY ./setup-localstack.sh /
RUN chmod +x /setup-localstack.sh
