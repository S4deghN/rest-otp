# An example OTP service exposed via a REST API

This project implements a simple OTP-based authentication system with a REST API.
Users can request a one-time password (OTP) and use it to log in (receive JWT).


# Prerequisites

* Go 1.20+
* Docker


# Run locally

## Setup database

The data base is set up inside a docker image anyways.

```sh
$ cd docker
$ docker compose up db
```

## Run app

```sh
$ cd src
$ go run main.go
```


# Run the whole app in docker

```sh
$ cd docker
$ docker compose up
```


# Usage

Server runs on `:8080` by default.

## Request OTP

```sh
$ curl -X POST http://localhost:8080/auth/otp-request \
     -H "Content-Type: application/json" \
     -d '{"phone":"1234567890"}'
```

## Login

```sh
$ curl -X POST http://localhost:8080/auth/login \
     -H "Content-Type: application/json" \
     -d '{"phone":"1234567890","otp":"123456"}'
```

## Query users

```sh
$ curl "http://localhost:8080/admin/users?limit=10"
```


# API Documentation
[View API Docs with Redoc](https://redocly.github.io/redoc/?url=https://raw.githubusercontent.com/S4deghN/rest-otp/refs/heads/master/api.yaml)

