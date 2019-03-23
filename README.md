# Indy

This is the source code for [https://indycar.xyz](https://indycar.xyz).

The site is a news aggregator I use for parse RSS-feeds and select news that 
match articles mentioning IndyCar, Marcus Ericsson or Felix Rosenqvist.

You can use this code to match some other words. Just change the `matchingArticle()` 
function in [pkg/news/news.go](pkg/news/news.go) to match something else.

## How to
1. `go get -u github.com/RasmusLindroth/indy`
2. Create a database and table named `indy`. Check [table.sql](./table.sql).
If you want to use another name  you'll have to change the code in
[pkg/database/database.go](pkg/database/database.go).
3. You should propably remove/change my google tracking code in [index.gohtml](./webfiles/templates/index.gohtml).
4. Create a [config-file](./config-sample.yml).
5. Install: `go install`
6. Run it: `./indy -conf ./config.yml`

## Requirements
* Go (some later version supporting modules)
* MySQL or MariaDB
