# TMS POC

## セットアップ

### 前提条件
- Docker
- Docker Compose

### データベースの起動

1. 環境変数ファイルを作成:
```bash
cp .env.example .env
```

2. Docker Composeでデータベースを起動:
```bash
docker compose up -d
```

3. データベースの状態確認:
```bash
docker compose ps
docker compose logs postgres
```

### データベース接続情報

- ホスト: `localhost`
- ポート: `5432`
- データベース名: `tms_db`
- ユーザー名: `postgres`
- パスワード: `postgres`

### SQLクライアントでの接続

以下のようなSQLクライアントから接続できます:
- DBeaver
- pgAdmin
- TablePlus
- psql (コマンドライン)

psqlでの接続例:
```bash
docker compose exec postgres psql -U postgres -d tms_db
```

または、ホストから直接:
```bash
psql -h localhost -p 5432 -U postgres -d tms_db
```

### データベースの停止

```bash
docker compose down
```

データを削除する場合:
```bash
docker compose down -v
```

## SQLファイルの実行

### 方法1: ヘルパースクリプトを使用（推奨）

1. `sql/` ディレクトリにSQLファイルを作成:
```bash
# 例: sql/001_create_tables.sql
```

2. SQLファイルを実行:
```bash
./run-sql.sh sql/001_create_tables.sql
```

### 方法2: 直接実行

```bash
docker compose exec -T postgres psql -U postgres -d tms_db < sql/001_create_tables.sql
```

### 方法3: psqlコマンドで実行

```bash
docker compose exec postgres psql -U postgres -d tms_db -f /path/to/file.sql
```

### 方法4: 対話モードで実行

```bash
# psqlに接続
docker compose exec postgres psql -U postgres -d tms_db

# psql内でファイルを実行
\i /path/to/file.sql

# または直接SQLを実行
CREATE TABLE ...;
```

## 初期SQLスクリプト

`init/` ディレクトリにSQLファイルを配置すると、コンテナ初回起動時に自動実行されます。
既に起動済みの場合は、上記の「SQLファイルの実行」方法を使用してください。
