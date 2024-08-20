# Library Management System

## Services

### Auth Service

- Login POST /v1/auth/login
- Register POST /v1/auth/register
- RefreshToken POST /v1/auth/refgresh

### Book Service

- ListBook GET /v1/book
- GetBook GET /v1/book/{id}
- CreateBook POST /v1/book/{id}
- UpdateBook PUT /v1/book/{id}
- DeleteBook DELETE /v1/book/{id}
- BorrowBook POST /v1/book/borrow/{id}
- ReturnBook POST /v1/book/return/{id}

### Author Service

## Development

### Manage Proto File

- Put proto file in `api/proto`
- Set package name with `github.com/purnasatria/library-management/api/<service_name>`
- Generate proto file using this command

  ```sh
  make generate-proto SCV=<service_name>
  ```

- It will generate go file on `api/gen/<service_name>`
  - if error `folder not available`, create it
- It will also generate swagger spec file on `api/swagger`

### Manage Migration

#### Create Migration

using make command

```sh
make create-migration SVC=<service_name> SEQ=<migration_name>
```
