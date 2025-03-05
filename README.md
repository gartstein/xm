# XM Golang Microservice

## Overview
This project is a microservice built in **Golang** for handling **companies**. It was developed as part of an **interview technical exercise** and follows production-ready best practices.

The service provides **CRUD** operations with **authentication and event-driven capabilities**.

## Technical Requirements

The microservice supports the following operations:
- **Create** a company
- **Update (Patch)** company details
- **Delete** a company
- **Get (Retrieve)** company details

### **Company Attributes**
Each company includes:
- **ID (UUID)** - Required
- **Name** (max 15 characters) - **Required & Unique**
- **Description** (max 3000 characters) - Optional
- **Employees Count (int)** - Required
- **Registered (boolean)** - Required
- **Type** (Corporation | NonProfit | Cooperative | Sole Proprietorship) - **Required**

### **Security**
- Only **authenticated users** can create, update, or delete companies.
- **JWT Authentication** is implemented.

## Features
✅ **Golang** backend with `gorm` for database operations  
✅ **gRPC-based** microservice with `protobuf` support  
✅ **PostgreSQL** as the database  
✅ **Kafka** for event-driven processing  
✅ **Dockerized** setup for easy deployment  
✅ **Linter and Code Quality Checks**  
✅ **Automated tests**  

## **Authentication Service**
This project includes a **mock authentication microservice** that generates **JWT tokens** for API authentication.

### **Features**
✅ Generates JWT tokens for authenticated users  
✅ Implements a simple `GET /token` endpoint  
✅ Dockerized for easy deployment  
✅ Uses **HS256 JWT signing method**

### **Running the Authentication Service**
You can run the authentication service independently:

#### **Using Docker Compose**
```sh
docker-compose up --build authentication
````
## Getting Started

### **Prerequisites**
Ensure you have the following installed:
- **Go** (`>=1.18`)
- **Docker & Docker Compose**
- **Make**
- **Protobuf Compiler (`buf`)**
- **golangci-lint**

### **Installation**
Clone the repository:
```sh
git clone https://github.com/gartstein/xm.git
cd xm
```

## Working with the Project (Makefile)
The project includes a `Makefile` that simplifies common development tasks.

### **Protobuf Commands**
- **Generate gRPC Protobuf Stubs:**
  ```sh
  make proto
  ```
- **Lint Protobuf Files:**
  ```sh
  make proto-lint
  ```
- **Clean Generated Protobuf Files:**
  ```sh
  make proto-clean
  ```

### **Golang Development**
- **Run Linter:**
  ```sh
  make lint
  ```
- **Build the Go Binary:**
  ```sh
  make build
  ```
- **Run Tests:**
  ```sh
  make test
  ```
- **Clean Build Artifacts:**
  ```sh
  make clean
  ```

### **Docker Commands**
- **Build Docker Image:**
  ```sh
  make docker-build
  ```
- **Run Services via Docker Compose (PostgreSQL, Kafka, gRPC service, etc.):**
  ```sh
  make docker-run
  ```

---

## Accessing the API

### **Running the Service**
To start the service locally, run:
```sh
make run-backend
```
or using Docker:
```sh
make docker-run
```

By default, the API runs on `http://localhost:8080`.

### **Authentication**
This API uses **JWT authentication**. Before calling secured endpoints, you must **obtain a token** using the login endpoint:

#### **1. Obtain a JWT Token**
```sh
curl -X POST http://localhost:8080/auth/login   -H "Content-Type: application/json"   -d '{
    "username": "admin",
    "password": "password"
  }'
```
The response will contain a JWT token, which you must include in all requests to protected endpoints.

---

### **API Endpoints**
#### **1. Create a Company**
```sh
curl -X POST http://localhost:8080/companies   -H "Authorization: Bearer <YOUR_TOKEN>"   -H "Content-Type: application/json"   -d '{
    "name": "My Company",
    "description": "A great company",
    "employees": 50,
    "registered": true,
    "type": "Corporation"
  }'
```

#### **2. Get a Company by ID**
```sh
curl -X GET http://localhost:8080/companies/<COMPANY_ID>   -H "Authorization: Bearer <YOUR_TOKEN>"
```

#### **3. Update a Company**
```sh
curl -X PATCH http://localhost:8080/companies/<COMPANY_ID>   -H "Authorization: Bearer <YOUR_TOKEN>"   -H "Content-Type: application/json"   -d '{
    "name": "Updated Company Name"
  }'
```

#### **4. Delete a Company**
```sh
curl -X DELETE http://localhost:8080/companies/<COMPANY_ID>   -H "Authorization: Bearer <YOUR_TOKEN>"
```

#### **5. Check if a Company Exists by Name**
```sh
curl -X GET http://localhost:8080/companies/exists?name=MyCompany   -H "Authorization: Bearer <YOUR_TOKEN>"
```

## Accessing the API via gRPC ##

#### 
### This service also supports gRPC in addition to HTTP. The gRPC API allows clients to communicate efficiently using protocol buffers.

---

## Expectations
This project was built as part of an **interview project** and follows **best practices** for production readiness.

The following **bonus features** are included:
- **Kafka event production** for mutating operations
- **Dockerized application**
- **gRPC-based API**
- **Integration tests**
- **Configuration file support**

## Conclusion
This microservice is designed to be **scalable, maintainable, and production-ready**. It follows **modern Golang development practices**, integrates **Kafka for event processing**, and provides **robust authentication**.

---

### 📌 **Next Steps**
- Run `make proto` before starting the service to generate protobuf files.
- Use `make test` to validate changes before committing.

---

## **License**
This project is released under the **MIT License**.
