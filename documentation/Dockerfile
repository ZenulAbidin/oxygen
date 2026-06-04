FROM nginx:1.27-alpine

COPY documentation/nginx.conf /etc/nginx/conf.d/default.conf
COPY documentation /usr/share/nginx/html
COPY web/redoc/redoc.standalone.js /usr/share/nginx/html/assets/redoc.standalone.js
COPY api/proto /usr/share/nginx/html/specs
