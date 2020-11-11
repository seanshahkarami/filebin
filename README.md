# filebin

Simple HTTP immutable file sharing service.

## Building

```sh
go build
```

## Usage

First, run the filebin server.

```sh
./filebin
```

Now, you can upload data using a simple HTTP POST.

```sh
curl -X POST localhost:8000/data/my-important-file -d'some really important data'
```

Others can now see this data at the URL you used.

```sh
curl localhost:8000/data/my-important-file
```
