# polyglot
Script to import problems from polygon.codeforces.com

If you have a polygon problem on your computer, you can run

```
EOLYMP_USERNAME=[username] EOLYMP_PASSWORD=[password] \
go run ./cmd/eolymp-polyglot ip ~/a/b/problem
```

Or you can directly download it from polygon

```
EOLYMP_USERNAME=[username] EOLYMP_PASSWORD=[password] \
POLYGON_LOGIN=[username] POLYGON_PASSWORD=[password] \
go run ./cmd/eolymp-polyglot dp https://polygon.codeforces.com/aaaaaa/tsypko/problem
```
