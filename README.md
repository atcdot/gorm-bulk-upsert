# Gorm Bulk Upsert

`Gorm Bulk Upsert` is a library to implement bulk `INSERT ... ON DUPLICATE KEY UPDATE` using [gorm](https://github.com/jinzhu/gorm). Execute bulk upsert just by passing a slice of struct, as if you were using a gorm regularly.

*Inspired by [gorm-bulk-insert](https://github.com/t-tiger/gorm-bulk-insert)*

## Purpose

When saving a large number of records in database, inserting at once - instead of inserting one by one - leads to significant performance improvement. This is widely known as bulk insert.

Gorm is one of the most popular ORM and contains very developer-friendly features, but bulk insert is not provided.

This library is aimed to solve the bulk insert problem.

## Installation

`$ go get github.com/atcdot/gorm-bulk-upsert`

This library depends on gorm, following command is also necessary unless you've installed gorm.

`$ go get github.com/jinzhu/gorm`

## Usage

```go
gormbulk.BulkUpsert(db, sliceValue, 3000)
```

Third argument specifies the maximum number of records to bulk upsert at once. This is because inserting a large number of records and embedding variable at once will exceed the limit of prepared statement.

Depending on the number of variables included, 2000 to 3000 is recommended.

```go
gormbulk.BulkUpsert(db, sliceValue, 3000, "Name", "Email")
```

Basically, upserting struct values are automatically chosen. However if you want to exclude some columns explicitly, you can specify as argument.

In the above pattern `Name` and `Email` fields are excluded.

### Feature

- Just pass a slice of struct as using gorm normally, records will be created.
    - **NOTE: passing value must be a slice of struct. Map or other values are not compatible.**
- `CreatedAt` and `UpdatedAt` are automatically set to the current time.
- Fields of relation such as `belongsTo` and `hasMany` are automatically excluded, but foreignKey is subject to Insert.

## Example

```go
package main

import (
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/atcdot/gorm-bulk-upsert"
	"log"
	"time"
	_ "github.com/jinzhu/gorm/dialects/mysql"
)

type fakeTable struct {
	ID        int
	Name      string
	Email     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func main() {
	db, err := gorm.Open("mysql", "mydb")
	if err != nil {
		log.Fatal(err)
	}

	var upsertRecords []interface{}
	for i := 0; i < 10; i++ {
		upsertRecords = append(upsertRecords,
			fakeTable{
				Name:  fmt.Sprintf("name%d", i),
				Email: fmt.Sprintf("test%d@test.com", i),
				// you don't need to set CreatedAt, UpdatedAt
			},
		)
	}

	err = gormbulkups.BulkUpsert(db, upsertRecords, 3000)
	if err != nil {
		// do something
	}

	// columns you want to exclude from Upsert, specify as an argument
	err = gormbulkups.BulkUpsert(db, upsertRecords, 3000, "Email")
        if err != nil {
            // do something
        }
}
```

## License

This project is under Apache 2.0 License. See the [LICENSE](https://github.com/kabukikeiji/gorm-bulk-insert/blob/master/LICENSE.txt) file for the full license text.


[gorm-bulk-upsert]: https://github.com/atcdot/gorm-bulk-upsert