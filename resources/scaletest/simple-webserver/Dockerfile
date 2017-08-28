FROM alpine

EXPOSE 9090

COPY simple-webserver /root/simple-webserver
RUN chmod a+x /root/simple-webserver
ENTRYPOINT ["/root/simple-webserver"]