import React from "react";
import { render } from "ink";
import App from "./App.js";

const enginePath = process.env["GOCLI_ENGINE_PATH"] ?? "gocode-engine";
const model = process.env["GOCLI_MODEL"] ?? "anthropic/claude-sonnet-4-20250514";
const mode = process.env["GOCLI_MODE"] ?? "plan";

render(<App enginePath={enginePath} model={model} mode={mode} />);
