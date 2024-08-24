# Library Management System

## 1. Overview

The Library Management System is a microservices-based application designed to manage various aspects of a library, including user authentication, book management, author information, and category organization.

## 2. General Architecture

The system is built on a microservices architecture, consisting of four main services:

1. **Auth Service**: Handles user authentication and authorization.
2. **Author Service**: Manages author information.
3. **Category Service**: Deals with book categories and classifications.
4. **Book Service**: Manages book information and operations.

Each service is containerized using Docker and orchestrated with Docker Compose. The services communicate with each other using gRPC for internal operations and expose REST APIs for external interactions. PostgreSQL is used as the database, with each service having its own database to ensure data isolation.

## 3. API and RPC Method List

### Auth Service

| Method       | gRPC           | REST                         | Description                  |
| ------------ | -------------- | ---------------------------- | ---------------------------- |
| Register     | `Register`     | POST `/api/v1/auth/register` | User registration            |
| Login        | `Login`        | POST `/api/v1/auth/login`    | User login                   |
| RefreshToken | `RefreshToken` | POST `/api/v1/auth/refresh`  | Refresh authentication token |
| VerifyToken  | `VerifyToken`  | POST `/api/v1/auth/verify`   | Verify authentication token  |

### Author Service

| Method       | gRPC           | REST                          | Description               |
| ------------ | -------------- | ----------------------------- | ------------------------- |
| CreateAuthor | `CreateAuthor` | POST `/api/v1/authors`        | Create a new author       |
| GetAuthor    | `GetAuthor`    | GET `/api/v1/authors/{id}`    | Retrieve author details   |
| UpdateAuthor | `UpdateAuthor` | PUT `/api/v1/authors/{id}`    | Update author information |
| DeleteAuthor | `DeleteAuthor` | DELETE `/api/v1/authors/{id}` | Delete an author          |
| ListAuthors  | `ListAuthors`  | GET `/api/v1/authors`         | List all authors          |

### Category Service

| Method                  | gRPC                      | REST                                     | Description                        |
| ----------------------- | ------------------------- | ---------------------------------------- | ---------------------------------- |
| CreateCategory          | `CreateCategory`          | POST `/api/v1/categories`                | Create a new category              |
| GetCategory             | `GetCategory`             | GET `/api/v1/categories/{id}`            | Retrieve category details          |
| UpdateCategory          | `UpdateCategory`          | PUT `/api/v1/categories/{id}`            | Update category information        |
| DeleteCategory          | `DeleteCategory`          | DELETE `/api/v1/categories/{id}`         | Delete a category                  |
| ListCategories          | `ListCategories`          | GET `/api/v1/categories`                 | List all categories                |
| UpdateItemCategories    | `UpdateItemCategories`    | PUT `/api/v1/items/{item_id}/categories` | Update categories for an item      |
| BulkAddItemToCategories | `BulkAddItemToCategories` | POST `/api/v1/categories/bulk-add-item`  | Add an item to multiple categories |
| GetItemCategories       | `GetItemCategories`       | GET `/api/v1/items/{item_id}/categories` | Get categories for an item         |

### Book Service

| Method                 | gRPC                     | REST                                     | Description                  |
| ---------------------- | ------------------------ | ---------------------------------------- | ---------------------------- |
| CreateBook             | `CreateBook`             | POST `/api/v1/books`                     | Create a new book            |
| GetBook                | `GetBook`                | GET `/api/v1/books/{id}`                 | Retrieve book details        |
| UpdateBook             | `UpdateBook`             | PUT `/api/v1/books/{id}`                 | Update book information      |
| DeleteBook             | `DeleteBook`             | DELETE `/api/v1/books/{id}`              | Delete a book                |
| ListBooks              | `ListBooks`              | GET `/api/v1/books`                      | List all books               |
| BorrowBook             | `BorrowBook`             | POST `/api/v1/books/{id}/borrow`         | Record a book being borrowed |
| ReturnBook             | `ReturnBook`             | POST `/api/v1/books/{id}/return`         | Record a book being returned |
| GetBookRecommendations | `GetBookRecommendations` | GET `/api/v1/books/{id}/recommendations` | Get book recommendations     |

## 4. Swagger Documentation

The Swagger JSON files are located at `/api/swagger/<service_name>.swagger.json`
Swagger JSON file is generated from the proto file that we define.

Swagger UI is available for each service to provide interactive API documentation. Once the services are running, you can access the Swagger UI at `<sevice_host>:<service_port>`

## 5. Handling Race Conditions and Ensuring Data Consistency

This system implements several strategies to handle race conditions and ensure data consistency across concurrent operations.

### Database-Level Handling

1. **Connection Pooling**:

   - We use Go's `sql.DB` to manage a pool of database connections.
   - Configuration is done in the `database.NewConnection` function:
     ```go
     db.SetMaxOpenConns(cfg.MaxOpenConns)
     db.SetMaxIdleConns(cfg.MaxIdleConns)
     db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
     ```

2. **FOR UPDATE SKIP LOCKED**:

   - Used in critical sections like book borrowing to handle concurrent updates.
   - Example in `BorrowBook`:
     ```go
     query := `
         UPDATE books
         SET available_copies = available_copies - 1
         WHERE id = $1 AND available_copies > 0
         RETURNING available_copies
         FOR UPDATE SKIP LOCKED
     `
     ```

3. **Transactions**:
   - All critical operations are wrapped in database transactions to ensure atomicity.

### Future Considerations for Complex Scenarios

As the system grows and becomes more complex, we may need to implement more robust solution to handle distributed transactions and ensure consistency across multiple services. Here are some approaches that could be considered:

- Saga Pattern:

  - For operations spanning multiple services, we could implement the Saga pattern.
  - Each step in a distributed transaction would have a corresponding compensating action to roll back changes if a step fails.

- Event-Driven Architecture:
  - Implement an event bus or message queue (e.g., RabbitMQ, Apache Kafka) for asynchronous communication between services.
  - Services would publish events when their state changes and subscribe to relevant events from other services.

## 6. Security Management

The Library Management System implements several security measures to protect data and ensure secure communication between services and clients.

1. **JWT for REST API**:

   - JSON Web Tokens (JWT) are used for authenticating REST API requests.
   - The Auth Service generates JWTs upon successful login.
   - Each JWT contains encoded user information and an expiration time.
   - Other services validate the JWT for protected endpoints.

2. **Server Key for Inter-Service Communication**:
   - A pre-shared server key is used to authenticate gRPC calls between services.
   - Each service includes this key in its gRPC metadata for outgoing calls.
   - Receiving services validate this key before processing the request.

## 7. Extending the Codebase

To add a new service or extend existing ones:

1. Create a new directory under `cmd/` for your service.
2. Define the service's proto file in `api/proto/`.
3. Generate the gRPC and REST gateway code using the `make proto-generate SVC=your_service_name` command.
4. Implement the service logic in Go by write the code in `internal/<service_name>`
5. Add the service to the `docker-compose.yml` file.
6. Update the Makefile if necessary to include build and run commands for new service.

## 8. Running the Application

### Prerequisites

- Docker and Docker Compose
- Go 1.21 or later
- Make

### Steps to Run

1. Clone the repository:
2. Set up environment variables:
   ```sh
   cp .env.example .env
   ```
   Edit the `.env` file with your configuration.
3. Build and start the services:
   ```sh
   make up
   ```
4. Run database migrations:
   ```sh
   make migrate-up
   ```
5. The services should now be running and accessible at their respective ports.

### Useful Commands

- Build all services: `make build`
- Generate proto files: `make proto-generate SVC=service_name`
- Create a new migration: `make create-table SVC=service_name SEQ=migration_name`
- Stop all services: `make down`

Refer to the Makefile for more available commands.
