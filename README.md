# OpenTelemetry for Go Developers: A Gentle Introduction

[Presentation](https://docs.google.com/presentation/d/1ufmrOUlN1Sbbj9ukxY2IdPL-2IDZTypAkSEeQ7eltJo/edit?usp=sharing)

Requirements:

- Go 1.20
- Docker Compose v2

Build and run the backend:

```sh
docker compose up -d
go install todo/cmd/todoservice
todoservice
```

Build and use the CLI app:

```sh
go install todo/cmd/todo
todo
todo add "important work"
todo list
todo add "very long description that is extremely important"
```
