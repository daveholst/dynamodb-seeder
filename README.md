# dynamodb-seeder
Tool to seed a dynamodb database. Currently working locally with dynamodb-local. Currently only offers an opinionated single table focused schema that alligns with another project. 

## Local Dev
1. Get `dynamodb-local` stack up with `docker-compose up -d`
2. Check the db is working with the included Web GUI at http://localhost:8001
3. Fire a table build and seed with `go run ./src/main.go`

## TODO
- [ ] Takes Optional Args
    - [x] Table Name `-n`
    - [x] Endpoint `-e`
    - [ ] Table Schema `-s`
    - [ ] Seed Data JSON file `-f`
- [ ] Works with an AWS dynamodb - this should kinda just work :shrug: