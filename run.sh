#/bin/sh

WD=$(dirname $0)
cd $WD
. env.sh
go run ./cmd
