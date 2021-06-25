# gomi

gomi, short from(go migrate) is a very minimalistic tool, which aims to do one thing only: execute database migrations. A database migration is in form of
**plain SQL**. gomi executes the plain sql using the specified driver in the config file. The config file is in json format and it should
contain all needed properties to build valid DSN.

```
$> gomi
Usage: gomi -config=CONFIG_FILE [ACTION...]
-config                 configuration file for connecting to database
ACTIONS:
-init                   create migrations table into the specified database to track migrations
-migrate -all [-dir]    migrate all migrations from the specified directory [-dir].
                        By default the directory is 'migrations' in the current working directory.
-migrate -name [-dir]   migrate migration with name [-name] from directory [-dir].
                        By default the directory is 'migrations' in the current working directory.
-new -name [-dir]       create new migration [-name] in the specified directory [-dir].
                        By default the directory is 'migrations' in the current working directory.
-help                   print this message
```

## Supported drivers
* (mysql)[https://github.com/go-sql-driver/mysql]
* (postgresql)[https://github.com/jackc/pgx]
