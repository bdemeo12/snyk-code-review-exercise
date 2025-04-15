# npm dependency server

A web server that provides a basic HTTP api for querying the dependency
tree of an [npm](https://npmjs.org) package.

## Prerequisites

* [Go v1.16](https://golang.org/dl/)

## Getting Started

To install dependencies and start the server in development mode:

```sh
go run main.go
```

The server will now be running on an available port (defaulting to 3000) and
will restart on changes to the src files.

Then we can try the `/package` endpoint. Here is an example that uses `curl` and
`jq`, but feel free to use any client.

```sh
curl -s http://localhost:3000/package/react/16.13.0 | jq .
```

Most of the code is boilerplate; the logic for the `/package` endpoint can be
found in [src/package.ts](api/api.go), and some basic tests in
[test/package.test.ts](api/api_test.go)

You can run the tests with:

```sh
go test ...
```


## Commands To Run 

### ok
```sh
 curl -s http://localhost:3000/package/react/16.13.0 | jq .
```
### large package
```sh
 curl -s http://localhost:3000/package/express/4.21.2 | jq .
```
### massive package
```sh
http://localhost:3000/package/npm/11.0.0 | jq .
```
### circular dependencies
TrueColor has a dependency Term-ng: https://www.npmjs.com/package/trucolor?activeTab=dependencies
Term-ng has a dependency TrueColor: https://www.npmjs.com/package/term-ng?activeTab=dependencies
```sh
curl -s http://localhost:3000/package/trucolor/4.0.4 | jq .
```
### invalid naming convention
```sh
curl -s http://localhost:3000/package/@snyk/snyk-docker-plugin/6.15.2 | jq .
```
### Another Large Package
```sh
curl -s http://localhost:3000/package/snyk-docker-plugin/6.15.2 | jq .
```
### Not Valid
```sh
curl -s http://localhost:3000/package/undefined/undefined | jq .
```
### Not Valid
```sh
curl -s http://localhost:3000/package/gibberish1234/1.0.0 | jq .
```
