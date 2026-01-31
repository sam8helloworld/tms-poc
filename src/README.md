# TMS POC - Go Application

国際物流SCMプラットフォームのバックエンドアプリケーション

## ディレクトリ構造

```
src/
├── cmd/
│   └── api/              # APIサーバーのエントリーポイント
├── internal/
│   ├── domain/
│   │   └── model/        # ドメインモデル定義
│   ├── infrastructure/
│   │   └── database/     # データベース接続・設定
│   └── repository/       # データアクセス層
└── pkg/
    └── config/           # 設定管理
```

## 開発開始

1. 依存関係のインストール:
```bash
go mod tidy
```

2. ドメインモデルの定義:
`internal/domain/model/` 配下にモデルファイルを作成してください

3. データベース接続の設定:
`internal/infrastructure/database/` 配下に接続処理を実装してください
