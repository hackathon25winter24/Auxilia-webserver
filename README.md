# Auxilia Webserver

Auxilia Battle Prototype向けの、インメモリ・サーバー権威型ゲームサーバーです。

```sh
go run .
```

既定では `http://localhost:8081` で起動します。戦闘ルールは `internal/game` に分離され、HTTPやDBに依存しません。
