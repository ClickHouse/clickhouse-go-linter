## ClickHouse clickhouse-go linter

A linter for the [clickhouse-go](https://github.com/ClickHouse/clickhouse-go/tree/main) driver.

The linter detects 2 common implementation mistakes:
- forgetting to call `rows.Err()` after calling `rows.Next()`
- forgetting to call `defer batch.Close()` when using `Batch`

These implementation mistakes are often observed in codebases using the clickhouse-go driver. They result 
in incomplete data and hard-to-troubleshoot bugs.  
Find more details with code examples in the [rules section](#rules). 

## Usage 

### With go vet

```bash
# install
go install github.com/ClickHouse/clickhouse-go-linter@latest

# run in your project
go vet -vettool=$(which clickhouse-go-linter) ./...
# --> returns a non-zero exit code if a check does not pass  
```

### With golangci-lint
_COMING SOON_

### Standalone
```bash
go install github.com/ClickHouse/clickhouse-go-linter@latest

# run in your project
clickhouse-go-linter ./...
```

## Development
See [CONTRIBUTING.md](CONTRIBUTING.md).

## Rules

## 1. chrowserr
Detect when the `github.com/ClickHouse/clickhouse-go/v2/lib/driver` `Rows.Err()` call is missing after a call to `Rows.Next()`.

Calling `Rows.Err()` is mandatory (see [here](https://clickhouse.com/docs/integrations/go#copy-in-some-sample-code)).
Failing to handle `Rows.Err()` results in subtle, hard to troubleshoot errors: partial or inconsistent result are passed downstream.

Incorrect code: 

```go
parsedRows := []... 
rows, _ := conn.Query(ctx, "SELECT ...")
for rows.Next() {
   parsedRow, err = ....
   if err {
     return nil, err
   }
   append(rows, parsedRow)
}

// !!! MISSING rows.Err() handling before returning !!! 
// !!! incomplete results may be returned with no errors !!!  

return parsedRows, nil
```

<details><summary>Correct code</summary>

```go
parsedRows := []... 
rows, _ := conn.Query(ctx, "SELECT ...")
for rows.Next() {
   parsed, err = ....
   if err {
     return nil, err
   }
   append(parsedRows, parsed)
}

// !!! correct - check rows.Err() before returning !!!
if err := rows.Err(); err != nil {
  return nil, err
}

return parsedRows, nil
```

</details>

### Rule details

The linter goes through every function. 
If `rows.Next()` is called in a function, it looks for `rows.Err()`. If not found, an error is reported.
Variable re-assignments and intertwined variable are supported. See [testcases.go](passes/chrowserr/testdata/src/testcases/testcases.go).

There are some limitations:
- the linter does not check if the output of `.Err()` is effectively used.   
  For instance: `_ = rows.Err()` is not caught as an error. The assumption is this kind of code would be written on purpose and would not be a mistake.  
- the linter does not cross function block boundaries. If `.Next()` is called in a function and `.Err()` is called in a 
  closure (eg a closure created inside the function), the linter will not be able to associate the `Err()` and the `.Next()` calls. 
  (note: in most cases such pattern is a bad idea). See `nextAndErrNotInSameFunctionBlock` test case. 
- even if `.Err()` is called _at some point_, improperly returning before the call to `.Err()` is not detected. (Contrived) example:
  ```
  parsedRows = []... 
  rows, _ := conn.Query(ctx, "SELECT 1")
  for rows.Next() {
    parsedRow, err = ....
    if err {
      return nil, err
    }
    append(rows, parsedRow)
    }
  
    if len(parsedRows) = 1 {
       // !!! returns without checking rows.Err !!! 
       return parsedRows[0], nill 
    }

    if err := rows.Err(); err != nill {
       // correct error handling
    }
    
    return parsedRows[1], nil
  ```
  --> in this case the linter will not be able to detect that `.Err()` may be called too late. 


## 2. chbatchclose
Detect when a `github.com/ClickHouse/clickhouse-go/v2/lib/driver` `Batch` variable is used but no associated `defer batch.Close()` is called.

Calling `defer batch.Close()` is highly recommended (see [here](https://clickhouse.com/docs/integrations/go#batch-insert)).
While it is possible to not use this statement, code that do not use it often end up with subtle leaks, as the 
underlying batch connection may not be closed if `Batch.Append` or `Batch.Send` fail.
As the `defer batch.Close()` does not change the logic or performance of the program, this rule enforces its usage.

Incorrect code:

```go
batch, err := conn.PrepareBatch(ctx, "INSERT INTO example")
if err != nil {
    return err
}
// !!! missing defer batch.Close() !!!

for i := 0; i < 1000; i++ {
    err := batch.Append(i),
    if err != nil {
        return err  // !!! connection leak !!!
	}
}

return batch.Send() // !!! potential connection leak if Send fails !!!
```

<details><summary>Correct code</summary>

```go
batch, err := conn.PrepareBatch(ctx, "INSERT INTO example")
if err != nil {
    return err
}
defer batch.Close()

for i := 0; i < 1000; i++ {
    err := batch.Append(i),
    if err != nil {
        return err
    }
}

return batch.Send()
```

</details>

### Rule details

The linter goes through every function.
If a clickhouse driver `Batch` is instantiated and is not part of the values returned by the function, a `defer batch.Close()` must be found.
Also, assigning a `Batch` to the blank identifier `_` is flagged.  
Variable re-assignments and intertwined variable are supported. See [testcases.go](passes/chbatchclose/testdata/src/testcases/testcases.go).

There are some limitations:
- except for looking into defer blocks;, the linter does not cross function block boundaries. If a `Batch` variable is instantiated and `batch.Close()` is called in a
  closure inside the defer call, the linter will not be able to associate the `batch.Close()` to the variable.
  (note: in most cases such pattern is a bad idea). See `deferCloseIsInClosure` test case.
- `defer batch.Close()` must be called after checking that the `PrepareBatch` call returned no error.
   incorrect: 
   ```
   batch, err := conn.PrepareBatch(ctx, "INSERT INTO example")
   defer batch.Close()
   if err != nil {
		return
   }
   ```
   correct:
   ```
   batch, err := conn.PrepareBatch(ctx, "INSERT INTO example")
   if err != nil {
		return
   }
   defer batch.Close()
   ```
   This is not enforced by the linter.
- it is in theory possible to write correct code without using `defer batch.Close()`, by properly checking  
  error results after each call to `.Append()`, `.Send()`, etc... and calling `Close` or `Abort` accordingly. Because 
  writing such correct code is difficult and prone to regressions, and because `defer batch.Close` does not 
  change the code correctness and is easy to add, this linter enforces its usage.

