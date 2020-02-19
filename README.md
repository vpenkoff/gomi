# gomi

gomi, short from(go migrate) is a very minimalistic tool, which aims to do one thing only: execute database migrations. A database migration is in form of
**plain SQL**. gomi executes the plain sql using the specified driver in the config file. The config file is in json format and it should
contain all needed properties to build valid dsn. Currently only **mysql** is supported.

## Usage

```
$> ./gomi --help

Usage:
gomi init - init migrations table
gomi migrate -new [name] - create migration with name
gomi migrate -all - migrate all migrations
gomi migrate -name [name] - migrate migration with name
```

* gomi init - when executed, new db table is created, which tracks run migrations
	* gomi migrate -new *migration name* - when executed, new migration file is created under the *migrations* directory
* gomi migrate -all - when executed, run all migrations from the *migrations* directory (which are not already run)
	* gomi migrate -name *migration_name* - when executed, run the migration with name *migration_name*

## Supported drivers
	* [mysql](https://github.com/go-sql-driver/mysql)