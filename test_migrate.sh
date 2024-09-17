rm heimdalld
go build ./cmd/heimdalld
rm ./app/migrated_dump-genesis.json
./heimdalld migrate ./app/dump-genesis.json