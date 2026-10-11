FROM node:22-alpine

WORKDIR /app

COPY package*.json ./
RUN npm install

COPY . .

EXPOSE 4200

CMD ["npm", "start", "--", "--proxy-config", "proxy.conf.json", "--host", "0.0.0.0", "--port", "4200"]
