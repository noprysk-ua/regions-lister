# Overview

Regions Lister is a sample project that uses the [Management API Golang client](https://github.com/noprysk-ua/managementapisdk).

## Requirements

The only requirement is a SingleStore API key. To get an API key follow [this](https://docs.singlestore.com/managed-service/en/developer-resources/management-api.html) doc page. Once it's done export it.

```
export API_KEY="my-key"
```

## Running

The following command runs the project.

```bash
go run main.go
```

Also, you may run it using docker.

```bash
docker run -v $(pwd):/tmp/regions-lister --workdir /tmp/regions-lister -e API_KEY=${API_KEY} golang:1.17 go run main.go
```

## Resources

 * [SDK](https://github.com/singlestore-labs/singlestore-go)
 * [Documentation](https://docs.singlestore.com/managed-service/en/developer-resources/management-api.html)
