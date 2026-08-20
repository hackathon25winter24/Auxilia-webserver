# Auxilia Webserver

Auxilia Battle Prototype向けの、MariaDB永続化・サーバー権威型ゲームサーバーです。

## 保存されるデータ

- ゲストセッション（トークンはSHA-256ハッシュのみ保存）
- 3体の編成
- マッチング待機状態
- 試合状態と戦闘ログ
- 処理済みコマンドID

戦闘状態はJSONで保存しますが、更新はMariaDBトランザクションと行ロックで保護されます。Goの `map` にはセッションや試合を保持しません。

## Docker Composeで起動

```sh
cp .env.example .env
docker compose up -d --build
```

MariaDBテーブルは起動時に自動作成されます。既定のAPI URLは `http://localhost:8081` です。
手動で作成する場合は `migrations/001_init.sql` を利用できます。

## 既存MariaDBへ接続

次の環境変数を設定してサーバーを起動します。

```sh
PORT=8081
NS_MARIADB_HOSTNAME=127.0.0.1
NS_MARIADB_PORT=3306
NS_MARIADB_USER=auxilia_user
NS_MARIADB_PASSWORD=change-me
NS_MARIADB_DATABASE=auxilia_web
DB_MAX_OPEN_CONNS=5
ALLOWED_ORIGINS=https://hackathon25winter24.github.io
```

```sh
go run .
```

`PORT` はHTTPサーバー、`NS_MARIADB_PORT` はMariaDBのポートです。

## 低メモリ設定

ComposeではMariaDBのInnoDBバッファプールを64MB、最大接続を20に制限しています。Go側のDB接続は既定で最大5、アイドル2です。契約サーバーの既存MariaDBを利用する場合は、サーバー全体の設定を確認して調整してください。

期限切れゲストと古いコマンドは1時間ごとに削除し、終了後7日を過ぎた試合も削除します。
