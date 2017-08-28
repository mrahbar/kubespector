FROM alpine

EXPOSE 9090

COPY vegeta-server /root/vegeta-server
RUN chmod a+x /root/vegeta-server
ENTRYPOINT ["/root/vegeta-server"]