#!/bin/bash

# SQLファイルを実行するヘルパースクリプト
# 使い方: ./run-sql.sh <SQLファイルパス>

if [ -z "$1" ]; then
  echo "使い方: ./run-sql.sh <SQLファイルパス>"
  echo "例: ./run-sql.sh sql/001_create_tables.sql"
  exit 1
fi

SQL_FILE=$1

if [ ! -f "$SQL_FILE" ]; then
  echo "エラー: ファイル '$SQL_FILE' が見つかりません"
  exit 1
fi

echo "SQLファイルを実行中: $SQL_FILE"
docker compose exec -T postgres psql -U postgres -d tms_db < "$SQL_FILE"

if [ $? -eq 0 ]; then
  echo "✓ SQLファイルの実行が完了しました"
else
  echo "✗ SQLファイルの実行中にエラーが発生しました"
  exit 1
fi
