FROM nginx:1.19.5-alpine

COPY nginx/github.conf /etc/nginx/conf.d/vhost.conf
COPY certs/server.pem /server.pem
COPY certs/server.key /server.key
