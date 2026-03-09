# Contributing notes

Thank you for you interest in contributing.
This linter is under active development. 
Please open an issue if you don't find the information you are looking for.

## Local setup

```bash
make
```
This will run tests and build the project in bin/clickhouse-go-linter 

Details: 

```bash
make fmt
make vet
make test
make build
```

## Create a release
```
git tag vx.y.z
git push origin vx.y.z
```