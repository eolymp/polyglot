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

If you have downloaded the problem using this tool, you can also run this command in order to update the problem

```
go run ./cmd/eolymp-polyglot up https://polygon.codeforces.com/aaaaaa/tsypko/problem
```

If you have a contest on Polygon that you want to upload, you can run the following command

```
go run ./cmd/eolymp-polyglot ic 12345
```

Where 12345 is the id of the polygon contest. This command will upload the problems from the contest to the problem archive. If you want to update all the problem in a contest, you need to replace "ic" by "uc".

It is possible to import problem in "ejudge" format. For example, using the following command

```
go run ./cmd/eolymp-polyglot --format=ejudge ip ~/a/b/problem
```
