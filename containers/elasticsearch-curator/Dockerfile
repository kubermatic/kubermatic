FROM python:alpine

RUN pip install -U --quiet elasticsearch-curator==5.6.0

ENTRYPOINT [ "/usr/local/bin/curator" ]
