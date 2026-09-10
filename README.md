# Auxilia Webserver

Auxilia Battle Prototype向けの、MariaDB永続化・サーバー権威型ゲームサーバーです。

## NeoShowcaseへのデプロイ

`trap.show` へのデプロイでは、リポジトリ直下の `ns.yaml` に従って
`Dockerfile` がビルドされます。`docker-compose.yml` はローカル起動用であり、
NeoShowcaseの公開経路やMariaDBコンテナの起動には使用されません。

NeoShowcase側でMariaDBリソースを接続し、次の環境変数がコンテナに渡ることを
確認してください。

```sh
NS_MARIADB_HOSTNAME=...
NS_MARIADB_PORT=3306
NS_MARIADB_USER=...
NS_MARIADB_PASSWORD=...
NS_MARIADB_DATABASE=...
ALLOWED_ORIGINS=https://hackathon25winter24.github.io
```

HTTP待受ポートにはNeoShowcaseから渡される `PORT` を使用します。未指定の場合は
`8080` で起動します。

## 保存されるデータ

開発者用テストモードは、サーバーの環境変数 `TESTMODE_PASSWORD` で有効化します。
NeoShowcaseのサーバーアプリに設定してください。未設定・空文字の場合は利用できません。
フロント側の環境変数には設定しません。

エントランスで3体を選択し、Tキーを押しながら「対戦開始」をクリックすると
パスワード入力が表示されます。認証後はプレイヤー名に1・2を付けた試合内IDを作り、
同じ編成で両陣営を操作できます。手番終了で操作陣営が切り替わります。
「終了」または手番の120秒経過でエントランスへ戻ります。通常の勝敗確定時も終了します。
テスト試合は使用率・編成総数に加算されません。試合状態の `testOwnerId` に
認証したゲストIDを保存し、再読み込み後もテスト試合として扱います。

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

MariaDBテーブルは起動時に自動作成されます。既定のAPI URLは `http://localhost:8080` です。
手動で作成する場合は `migrations/001_init.sql` を利用できます。

Composeのappサービスには、`auxilia-web.trap.show` をコンテナ内部の
`8080`番ポートへ転送するtraefikラベルが設定されています。公開サーバー上で
別のルーター名や外部Dockerネットワークが指定されている場合は、その環境の
traefik設定に合わせてください。

## 既存MariaDBへ接続

次の環境変数を設定してサーバーを起動します。

```sh
PORT=8080
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
