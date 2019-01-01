# go-restapi-multitenant
An example on how to implement a REST API with JWT authorization and schema multitenancy using PostgreSQL

## This example uses the following libraries:
- Gorilla Mux: [link](https://github.com/gorilla/mux)
- GORM: [link](https://github.com/jinzhu/gorm)
- JWT-GO: [link](https://github.com/dgrijalva/jwt-go)

## The flow of a request

The service expects to receive a JWT token through the _Authorization_ header on every request. This token, once parsed, must have the claim '_tenant_' defined, which will be used as the schema name for persistence purposes.

Every request calls the __authorize__ function which is responsible for validating the JWT token and extracting the _claims_ from it, the __authorize__ function then calls the __prepare__ function passing the tenant name.

The __prepare__ function then executes the following tasks:
- Starts a new transaction
- Creates a schema for the tenant if it does not exists
- Switch the schema to the newly created one
- Migrate the database, creating the __person__ table if it not exists
- Calls the __handler__ to serve the request
- If the __handler__ returns any error, the transaction is rolled back, otherwise, the transaction is commited

The __handlers__ are the functions responsible to serve the requests:
- _ListPeople_ (Returns all persons)
- _GetPerson_ (Returns one person, through its ID)
- _CreatePerson_ (Creates a new person)
- _UpdatePerson_ (Update any data of an already created person)
- _DeletePerson_ (Deletes an previously created person)

All __handler__ functions respect the __handler__ type, so they implement the __authorize__ and __prepare__ functions by convention and can be instantiated into variables of type __handler__ as can be seen at the _main_ function


