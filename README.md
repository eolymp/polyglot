# polyglot
Script to import problems from polygon.codeforces.com

Before running, please copy and edit cmd/config/config-sample.yml to /cmd/config/config.yml

If you have a polygon problem on your computer, you can run

```
go run ./cmd/eolymp-polyglot ip ~/a/b/problem
```

Or you can directly download it from polygon

```
go run ./cmd/eolymp-polyglot dp https://polygon.codeforces.com/aaaaaa/tsypko/problem
```

If you want to update a problem, you need to add --id=11111 before the command. For example,


```
go run ./cmd/eolymp-polyglot --id=11111 dp https://polygon.codeforces.com/aaaaaa/tsypko/problem
```
