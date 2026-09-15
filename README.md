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

## 週間キャラクター使用状況API

以下は認証不要の読み取りAPIです。従来の `/api/characters` の累計値は変更しません。

| エンドポイント | 内容 |
| --- | --- |
| `GET /api/character-usage` | 今週の途中経過。`partial: true`、`recordingSince` は記録開始日時 |
| `GET /api/character-usage/history` | 完了した週の確定集計を古い順に返す |
| `GET /api/character-usage/counts.csv` | 週間使用数のCSV |
| `GET /api/character-usage/rates.csv` | 週間使用率のCSV（百分率、小数点以下4桁、%記号なし） |

集計期間は日本時間の月曜日00:00以上〜翌月曜日00:00未満です。
試合成立時ではなく、両プレイヤーが準備を完了して試合を開始した時点で
開始日時・両編成を記録します。使用率は「週間使用数 ÷ 週間編成総数 × 100」で、
通常対戦1試合につき編成総数は2です。テストモードは記録も集計もしません。

起動時と毎分、集計待ちの完了週を確認してDBへ保存します。週につき1回だけ保存され、
停止中に取り逃した集計は起動後に記録から補完します。履歴・CSV取得時にも未処理週を確認します。
試合開始の記録と集計確定を共通のDB行ロックで同期し、試合ID・対象週の主キーで重複を防ぎます。

CSVの先頭列は `week_start`（対象週の月曜日）、以降はIDのアルファベット順です。
新キャラクター追加前の週は空欄、存在していて使用されなかった場合は使用数0です。
対戦がない週の使用率はCSVでは空欄、JSONではnullです。
保存済み週の値は以降の累計値やキャラクター追加で変化しません。

導入前の履歴は復元しません。導入週は不完全なので確定履歴に含めず、翌月曜日からの
最初の完全な1週間が終了した時点で最初のCSV行ができます。今週の途中経過APIは導入直後から利用できます。
追加の環境変数や外部スケジューラーは不要です。起動時に追加テーブルを自動作成します。
週間集計と元の編成記録は、通常の7日後の試合削除とは独立して保持します。

## 低メモリ設定

ComposeではMariaDBのInnoDBバッファプールを64MB、最大接続を20に制限しています。Go側のDB接続は既定で最大5、アイドル2です。契約サーバーの既存MariaDBを利用する場合は、サーバー全体の設定を確認して調整してください。

期限切れゲストと古いコマンドは1時間ごとに削除し、終了後7日を過ぎた試合も削除します。

## アクティブプレイヤー数

- `POST /api/presence/heartbeat`：ゲスト認証必須。サーバー時刻で最終通知日時を更新し、`{"count": N}`を返します。
- `GET /api/presence/count`：直近60秒以内に通知したプレイヤーID数を返します。キャッシュしません。
- `DELETE /api/presence`：ゲスト認証必須。自身の接続記録を削除します。

ログイン中は画面を問わず20秒ごとに通知し、エントランスに `Active: N人` を表示します。同じIDの複数タブは1人、別IDは別人として数えます。タイトルに戻る際は削除し、切断時は60秒で集計対象外になります（画面への反映は次の通知時）。タブを再表示した際にも通知します。テストモードもログインしている実ユーザーのIDのみカウントします。

起動時のAutoMigrateで `web_presence` を作成します。24時間以上前の記録は既存の定期クリーンアップで削除します。DB共有により複数サーバーでも同じ集計になります。
