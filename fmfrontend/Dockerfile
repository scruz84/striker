FROM node:19-alpine as builder

WORKDIR /app
COPY . .

#https://stackoverflow.com/questions/52103155/reading-an-environment-variable-in-react-which-was-set-by-docker
ENV NODE_ENV production
RUN npm install --legacy-peer-deps
RUN npm run build

FROM nginx:1.23.4

COPY --from=builder /app/build /usr/share/nginx/html
COPY --from=builder /app/set-env.sh /usr/share/nginx/html
RUN chmod +x /usr/share/nginx/html/set-env.sh
COPY nginx.conf /etc/nginx/nginx.conf

EXPOSE 80

CMD ["sh", "-c", "cd /usr/share/nginx/html/ && ./set-env.sh && nginx -g 'daemon off;'"]