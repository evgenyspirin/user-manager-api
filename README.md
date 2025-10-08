#  User Manager API

### Introduction.
Dear reviewers team,

when I develop an application, first and foremost I pay particular attention to **performance**,
**resource efficiency**, **scalability**, and **fault tolerance**. My previous experience in high-load
projects has given me valuable experience in mastering all of these principles.
For this reason, I deliberately did not use the packages mentioned in the assignment requirements.

- **ORM** - https://github.com/go-gorm/gorm  
    The Go language was developed for high performance, and the approach to using ORM is, in a sense,  
    an anti-pattern in the context of highly efficient code. It contains many wrappers and reflections,  
    as well as often unnecessary allocations. As a result, we are not sure exactly what query was actually sent to the database.  
**Recommendation**: use pure SQL queries and work with the driver directly.  
- **logrus** - https://github.com/sirupsen/logrus  
Zap is much faster — it uses zero allocations and pre-serializes structured logs to JSON.  
Zap compiles log field definitions (zap.Field) instead of relying on runtime reflection.  
**Recommendation**: https://github.com/uber-go/zap
- **DB** – MySQL  
In the challenge I'm using PostgresSQL because it has many very useful features like:  
"RETURNING", "COALESCE", "bool", "IP address"... 
- **Storage** – AWS S3  
When saving files, I simulate saving and deleting files in cloud storage because I don't have my own rented server. 

#### Conclusion:
My comments are intended to provide **recommendations**, but if your project already uses certain technologies and they are not a bottleneck,
then it is reasonable to continue using them.

---

## Overview

The application is built on **The Twelve-Factor App** rules:

- **One code base** – GitHub
- **Clearly declared and isolated dependencies** – `go.mod`
- **Configuration** must be located in environment variables – `.env`
- **Strict separation of build, release, and execution** – CI/CD pipeline (future)
- **Stateless processes** – we store data in constant storage and update it (**PostgreSQL**)
- **Port binding** – the built-in web server runs on the specific port from the environment variable
- **Concurrency** – processes can be split into separate microservices in the future for high-load spots
- **Disposability** – supports graceful shutdown through a single `Context`
- **Logs** – currently using **Zap Logger**, future: **ELK Stack** (Elasticsearch + Logstash + Kibana)
- **Admin processes** – QA/Dev/Prod environments must be as similar as possible (future)

The application has a structure based on Go convention:
- https://github.com/golang-standards/project-layout  
Therefore "**rmqconsumer**" has external pkg.

Interfaces: “Go interfaces generally belong in the package that uses values of the interface type, not the package that implements those values.”
- https://go.dev/wiki/CodeReviewComments

---

## DDD Architectural Structure

The application uses **DDD (Domain Driven Design)** architecture with fully separated layers:

- **Interface** – REST, controllers, middlewares, HTTP request/response DTOs
- **Application** – use cases, calls domain objects and repositories through interfaces
- **Domain** – domain entities (currently no complex business rules)
- **Infrastructure** – repository implementations, DB models, storages (Redis, Postgres), clients (gRPC/HTTP), MQ...

**Request flow:**  
`client → interface → application → domain → infrastructure → DB`

**Dependencies** are directed inward: outer layers depend on inner ones, not vice versa.

---

## Concurrency Patterns

The application uses:
- **Fan-In** – for sending messages to a single chan from a different requests asynchronously

---

## API Specifications

External contracts follow **OpenAPI standards**:  
`internal/interface/api/rest/api-specs/openapi/usermanagerapi/openapi.yaml`

All possible cURL requests are located here and can be run directly from your IDE (tested in GoLand):  
`internal/interface/api/rest/api-specs/usermanagerapi.http`

---

## Tests

Covered the most important core logic.

- Pattern: **TableDrivenTests**
- Library: [`testify`](https://github.com/stretchr/testify)

Run from the root project directory to see code coverage:

```bash
$ go test ./... -coverprofile=coverage.out
$ go tool cover -html=coverage.out
```

---

## Ops

Infrastructure endpoints for metrics and health checks:

-- `http://localhost:8080/api/v1/metrics`:
* "usermanager_general_counters{result="app_requests_total"}" - total requests
* "usermanager_general_counters{result="user_created_total"}" - total created users 
* "usermanager_general_counters{result="user_updated_total"}" - total updated  users 
* "usermanager_general_counters{result="user_deleted_total"}" - total deleted  users 
* "usermanager_general_counters{result="user_files_created_total"}" - total created files 

-- `http://localhost:8080/api/v1/healthz`

Logs through middleware of request has info:

* Request Method
* Request Endpoint
* Request Code
* Request Duration
* Request Body

---

## Application Initialization Steps

1. Create application
2. Get configuration
3. Init logs, clients, DBs, etc.
4. Run application including all parallel processes:
    - HTTP server
    - `PublisherWorker` for asynchronous and parallel messages publishing into RabbitMQ
    - `DeliveryWorker` for asynchronous and parallel messages consuming from RabbitMQ
5. On `SIGURG` signal or context cancel, gracefully shut down the application

---

## RabbitMQ Web UI

-- `http://localhost:15672/`
- **login** – test
- **password** – test

---

## Using

Make sure that you have preinstall [`Docker`](https://www.docker.com/) on your OS.

To start local env run command from the root project dir:
```bash
$ docker compose up -d
```
For termination:
```bash
$ docker compose down -v
```

Since some our endpoints protected by auth(JWT+Middleware+Context)  
we have already existing Admin user. Use creds below for further login etc.   
- **email** – admin@example.com
- **password** – admin123  

Now see the section "API Specifications" above and have fun ;-)
