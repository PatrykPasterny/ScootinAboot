# Use a base image with Go installed
FROM golang:1.20-alpine

# Set the working directory inside the container
WORKDIR /app

# Copy the go.mod and go.sum files to the container
COPY go.mod go.sum ./

# Download Go dependencies
RUN go mod download
RUN wget -O /usr/local/bin/wait-for-it.sh https://raw.githubusercontent.com/vishnubob/wait-for-it/master/wait-for-it.sh
RUN chmod +x /usr/local/bin/wait-for-it.sh

# Copy the entire project to the container
COPY . .

# Build the Go binary
RUN go build -o myapp

# Expose the port on which your app listens
EXPOSE 8081

# Specify the command to run your app when the container starts
CMD ["./myapp"]