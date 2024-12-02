#!/bin/bash

SUPERVISOR_CONFIG="/etc/supervisor/conf.d/supervisord.conf"
NGINX_CONFIG="/etc/nginx/nginx.conf"


# Create an array of ports from the environment variable
IFS=',' read -r -a PORTS <<< "${GOTENBERG_PORTS:-3000,3001,3002,3003,3004}"

# Generate the Supervisor configuration file from the template
generate_supervisor_config() {
  echo "[supervisord]
nodaemon=true

[program:nginx]
command=/usr/sbin/nginx -g 'daemon off;'
autostart=true
autorestart=true
stdout_logfile=/var/log/supervisor/nginx.log
stderr_logfile=/var/log/supervisor/nginx_err.log
" > "$SUPERVISOR_CONFIG"

for index in "${!PORTS[@]}"; do
    port="${PORTS[$index]}"
    echo "[program:gotenberg$index]
command=/usr/bin/gotenberg --api-port $port --libreoffice-auto-start
autostart=true
autorestart=true
user=gotenberg
stdout_logfile=/var/log/supervisor/gotenberg$index.log
stderr_logfile=/var/log/supervisor/gotenberg$index_err.log
" >> "$SUPERVISOR_CONFIG"
done
}

# Generate the Nginx configuration file
generate_nginx_config() {
  echo "worker_processes 1;
events {
    worker_connections 1024;
}

http {
    upstream gotenberg_backend {
    least_conn;
" > "$NGINX_CONFIG"

for port in "${PORTS[@]}"; do
    echo "        server localhost:$port;" >> "$NGINX_CONFIG"
done

echo "    }

    server {
        listen 8080;
        client_max_body_size 26M;

        location / {
            proxy_pass http://gotenberg_backend;
            proxy_set_header Host \$host;
            proxy_set_header X-Real-IP \$remote_addr;
            proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto \$scheme;
        }
    }
}
" >> "$NGINX_CONFIG"
}

# Generate the configuration files
generate_supervisor_config
generate_nginx_config

supervisord -c /etc/supervisor/conf.d/supervisord.conf