# AITuber OnAir Puppet Controller

![AITuber OnAir Puppet Controller Logo](logo/aituber-onair-puppet-controller_logo.png)

## Overview
AITuber OnAir Puppet Controller provides a lightweight WebSocket relay and HTTP trigger endpoint for driving the [AITuber OnAir](https://aituberonair.com) client from external tools. It offers a Go implementation together with an integrated browser UI for direct control.

[日本語版](./README.ja.md)

## Features
- Broadcasts chat payloads to every connected WebSocket client via `/direct-speech`.
- Accepts JSON POST requests to `/trigger` and validates user input before broadcasting.
- Serves an embedded web UI that allows direct message submission from the browser.
- Uses a concurrency-safe hub abstraction to manage active WebSocket connections.

## Requirements
- Go 1.22 or newer

## Quick Start
1. Fetch dependencies (required only once):
   ```bash
   go mod tidy
   ```
2. Run the server:
   ```bash
   go run .
   ```
   > When working in a restricted environment you can set `GOCACHE=$(pwd)/.gocache` before `go run` to keep build artifacts inside the project folder.
3. Open the demo UI at [http://localhost:9000](http://localhost:9000) and connect your AITuber OnAir client to `ws://localhost:9000/direct-speech`.

## API
| Endpoint | Method | Description |
| --- | --- | --- |
| `/direct-speech` | GET (upgrade) | Upgrades to a WebSocket connection and receives broadcast messages. |
| `/trigger` | POST | Accepts `{ "text": string }`, validates input, and broadcasts the payload. |
| `/` | GET | Serves the embedded web UI for manual control. |
| `/status` | GET | Returns JSON with total, UI, and external WebSocket connection counts. |

## Development Notes
- The project relies on [github.com/gorilla/websocket](https://github.com/gorilla/websocket) for WebSocket handling.
- Logging is sent to stdout; you can pipe or redirect it to integrate with your preferred monitoring setup.
- To customize the HTML UI, edit `web/index.html`.

## Localization
A Japanese version of this document is available in [`README.ja.md`](README.ja.md).
