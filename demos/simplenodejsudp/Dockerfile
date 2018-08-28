FROM mhart/alpine-node:10.9.0
RUN mkdir -p /app
WORKDIR /app
COPY package.json /app
RUN npm install
COPY . /app
EXPOSE 22222
CMD ["node","index"]