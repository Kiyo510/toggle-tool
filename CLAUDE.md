# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## プロジェクト概要

Toggl Track APIを使用してタイムエントリデータを取得し、日付ごとにタグ別で集計したデータをテーブル形式で標準出力するGoアプリケーション。土日祝日は集計結果から除外される。

## 開発コマンド

### 実行
```bash
go run cmd/main.go
```

### ビルド
```bash
go build -o toggle-tool cmd/main.go
```

### 依存関係管理
```bash
go mod tidy
go mod download
```

## 環境設定

以下の環境変数を`.env`ファイルに設定する必要がある：

- `WORKSPACE_ID`: Toggl ワークスペースID
- `TOGGLE_API_KEY`: Toggl API キー

## アーキテクチャ

### 単一ファイル構成
現在のアーキテクチャは`cmd/main.go`に全機能が集約されている：

1. **データ構造**: Toggl API用のリクエスト/レスポンス構造体
2. **認証**: HTTPトランスポートでAPI認証を実装
3. **データ処理**: タイムエントリのタグ別・日付別集計
4. **フィルタリング**: 土日祝日除外（jpholidayライブラリ使用）
5. **表示**: tablewriterライブラリによるテーブル出力

### 主要ライブラリ
- `github.com/joho/godotenv`: 環境変数読み込み
- `github.com/najeira/jpholiday`: 日本の祝日判定
- `github.com/olekukonko/tablewriter`: テーブル表示

## 開発上の注意点

### テスト
現在テストファイルが存在しない。TDD原則に従い、新機能追加時はテストを先に作成する。

### 日付設定
コード内で対象日付がハードコードされている（2025年5月）。任意の日付設定機能を追加する場合はこの部分を修正する。

### データ取得
Toggl APIからのデータ取得はPageSize=3000で全データを一度に取得する設計になっている。

### タイムゾーン
Asia/Tokyoタイムゾーンを使用して日本時間での処理を行う。